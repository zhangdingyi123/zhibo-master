package ws

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/zhibo/backend/internal/domain"
	"github.com/zhibo/backend/internal/infra/metrics"
	redisc "github.com/zhibo/backend/internal/infra/redis"
	"github.com/zhibo/backend/internal/repository"
	"github.com/zhibo/backend/internal/service"
)

// Hub 按 roomId 隔离的 WebSocket 房间网关（4.1）
// Kafka（或 Redis Pub/Sub 降级）跨实例广播；倒计时 tick 仍本机推送
type Hub struct {
	rooms    map[string]*room
	mu       sync.RWMutex
	sessions *repository.SessionRepo
	bids     *repository.BidRepo
	snap     *service.UserAuctionService
	bidSvc   *service.BidService
	limiter  *bidRateLimiter
	redis    *redisc.Client
	mq       RoomBroadcaster

	registerCh   chan *Client
	unregisterCh chan *Client
	broadcast    chan roomBroadcast

	broadcastCancel context.CancelFunc
}

type room struct {
	id      string
	clients map[*Client]bool
	store   *EventStore
	mu      sync.RWMutex
}

type roomBroadcast struct {
	roomID string
	data   []byte
}

// NewHub 创建 WS 网关；mq 为 Kafka 广播；无 mq 时可用 Redis Pub/Sub 降级
func NewHub(
	sessions *repository.SessionRepo,
	bids *repository.BidRepo,
	snap *service.UserAuctionService,
	bidSvc *service.BidService,
	rdb *redisc.Client,
	mq RoomBroadcaster,
) *Hub {
	h := &Hub{
		rooms:      make(map[string]*room),
		sessions:   sessions,
		bids:       bids,
		snap:       snap,
		bidSvc:     bidSvc,
		redis:      rdb,
		mq:         mq,
		limiter:    newBidRateLimiter(300 * time.Millisecond),
		registerCh:   make(chan *Client),
		unregisterCh: make(chan *Client),
		broadcast:    make(chan roomBroadcast, 256),
	}
	go h.run()
	go h.runCountdownLoop()

	ctx, cancel := context.WithCancel(context.Background())
	h.broadcastCancel = cancel
	switch {
	case mq != nil:
		if err := mq.StartSubscriber(ctx, h.onCrossInstanceBroadcast); err != nil {
			log.Printf("ws: kafka subscriber: %v", err)
		} else {
			log.Printf("ws: kafka broadcast enabled (multi-instance)")
		}
	case rdb != nil:
		if err := rdb.StartRoomBroadcastSubscriber(ctx, h.onCrossInstanceBroadcast); err != nil {
			log.Printf("ws: redis broadcast subscriber: %v", err)
		} else {
			log.Printf("ws: redis pub/sub broadcast enabled (fallback)")
		}
	}
	return h
}

func (h *Hub) onCrossInstanceBroadcast(roomID string, envelopeJSON []byte) {
	h.deliver(roomID, envelopeJSON)
}

func parseRoomEventFromEnvelope(raw []byte) *RoomEvent {
	var env Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil
	}
	var ev RoomEvent
	if len(env.Payload) > 0 {
		_ = json.Unmarshal(env.Payload, &ev)
	}
	if ev.Seq == 0 && env.Seq > 0 {
		ev.Seq = env.Seq
	}
	if ev.Type == "" {
		return nil
	}
	return &ev
}

func (h *Hub) run() {
	for {
		select {
		case c := <-h.registerCh:
			h.joinRoom(c)
		case c := <-h.unregisterCh:
			h.leaveRoom(c)
		case b := <-h.broadcast:
			h.deliver(b.roomID, b.data)
		}
	}
}

func (h *Hub) getOrCreateRoom(roomID string) *room {
	h.mu.Lock()
	defer h.mu.Unlock()
	if r, ok := h.rooms[roomID]; ok {
		return r
	}
	r := &room{
		id:      roomID,
		clients: make(map[*Client]bool),
		store:   NewEventStore(defaultEventBuffer),
	}
	h.rooms[roomID] = r
	return r
}

func (h *Hub) joinRoom(c *Client) {
	r := h.getOrCreateRoom(c.roomID)
	r.mu.Lock()
	r.clients[c] = true
	r.mu.Unlock()
}

func (h *Hub) leaveRoom(c *Client) {
	h.mu.RLock()
	r, ok := h.rooms[c.roomID]
	h.mu.RUnlock()
	if !ok {
		return
	}
	r.mu.Lock()
	delete(r.clients, c)
	empty := len(r.clients) == 0
	r.mu.Unlock()
	if empty {
		h.mu.Lock()
		if len(r.clients) == 0 {
			delete(h.rooms, c.roomID)
		}
		h.mu.Unlock()
	}
}

func (h *Hub) deliver(roomID string, data []byte) {
	h.mu.RLock()
	r, ok := h.rooms[roomID]
	h.mu.RUnlock()
	if !ok {
		return
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for c := range r.clients {
		select {
		case c.send <- data:
		default:
		}
	}
}

func (h *Hub) registerClient(c *Client) {
	h.registerCh <- c
}

func (h *Hub) removeClient(c *Client) {
	h.unregisterCh <- c
}

// broadcastEphemeral 广播不计入 seq 的事件（如 countdown.tick）
func (h *Hub) broadcastEphemeral(roomID, eventType string, payload any) {
	ev := RoomEvent{
		Type:    eventType,
		Ts:      time.Now().UnixMilli(),
		Payload: marshalPayload(payload),
	}
	env := Envelope{
		Type:      ServerEvent,
		RoomID:    roomID,
		Timestamp: ev.Ts,
		Payload:   marshalPayload(ev),
	}
	b, _ := json.Marshal(env)
	h.broadcast <- roomBroadcast{roomID: roomID, data: b}
}

// Publish 向房间追加事件并广播（带 seq，供重连补偿）
func (h *Hub) Publish(roomID, eventType string, payload any) uint64 {
	ev := RoomEvent{
		Type:    eventType,
		Ts:      time.Now().UnixMilli(),
		Payload: marshalPayload(payload),
	}

	if h.redis != nil {
		ctx := context.Background()
		seq, err := h.redis.IncrRoomEventSeq(ctx, roomID)
		if err != nil {
			log.Printf("ws: incr room seq %s: %v", roomID, err)
			return 0
		}
		ev.Seq = seq
		env := Envelope{
			Type:      ServerEvent,
			RoomID:    roomID,
			Seq:       seq,
			Timestamp: ev.Ts,
			Payload:   marshalPayload(ev),
		}
		b, err := json.Marshal(env)
		if err != nil {
			return seq
		}
		if err := h.redis.StoreRoomEvent(ctx, roomID, b); err != nil {
			log.Printf("ws: store room event %s: %v", roomID, err)
		}
		if h.mq != nil {
			if err := h.mq.Publish(ctx, roomID, b); err != nil {
				log.Printf("ws: kafka publish %s: %v", roomID, err)
			}
		} else if err := h.redis.PublishRoomBroadcast(ctx, roomID, b); err != nil {
			log.Printf("ws: redis publish %s: %v", roomID, err)
		}
		// 投递由 Kafka / Redis 订阅回调 onCrossInstanceBroadcast 完成（含本实例）
		return seq
	}

	r := h.getOrCreateRoom(roomID)
	seq := r.store.Append(ev)
	env := Envelope{
		Type:      ServerEvent,
		RoomID:    roomID,
		Seq:       seq,
		Timestamp: ev.Ts,
		Payload:   marshalPayload(ev),
	}
	b, _ := json.Marshal(env)
	h.broadcast <- roomBroadcast{roomID: roomID, data: b}
	return seq
}

func (h *Hub) handleSubscribe(c *Client, payload SubscribePayload) {
	roomID := payload.RoomID
	if roomID == "" {
		roomID = c.roomID
	}
	if roomID == "" {
		c.sendJSON(Envelope{
			Type:    ServerError,
			Payload: marshalPayload(ErrorPayload{Code: 400, Message: "roomId required"}),
		})
		return
	}

	ctx := context.Background()
	session, err := h.sessions.GetByRoomID(ctx, roomID)
	if err != nil {
		c.sendJSON(Envelope{
			Type:    ServerError,
			Payload: marshalPayload(ErrorPayload{Code: 404, Message: "room not found"}),
		})
		return
	}

	if payload.ClientID != "" {
		c.clientID = payload.ClientID
	}
	c.roomID = roomID
	c.lastSeq = payload.LastSeq
	h.registerClient(c)

	h.getOrCreateRoom(roomID)
	var currentSeq uint64
	var missed []RoomEvent
	if h.redis != nil {
		currentSeq, _ = h.redis.CurrentRoomEventSeq(ctx, roomID)
		if payload.LastSeq > 0 && payload.LastSeq < currentSeq {
			rawList, err := h.redis.RoomEventsSince(ctx, roomID, payload.LastSeq)
			if err == nil {
				missed = roomEventsFromStored(rawList)
			}
		}
	} else {
		r := h.getOrCreateRoom(roomID)
		currentSeq = r.store.CurrentSeq()
		if payload.LastSeq > 0 && payload.LastSeq < currentSeq {
			missed = r.store.Since(payload.LastSeq)
		}
	}

	snap, err := h.snap.SnapshotByRoom(ctx, roomID)
	if err != nil {
		c.sendJSON(Envelope{
			Type:    ServerError,
			Payload: marshalPayload(ErrorPayload{Code: 404, Message: err.Error()}),
		})
		return
	}

	// 重连补偿（4.6）
	if len(missed) > 0 {
		c.sendJSON(Envelope{
			Type:     ServerSync,
			ClientID: c.clientID,
			RoomID:   roomID,
			Payload:  marshalPayload(SyncPayload{Snapshot: snap, Events: missed}),
		})
	}

	connPayload := ConnectedPayload{
		RoomID:     roomID,
		SessionID:  session.ID,
		CurrentSeq: currentSeq,
	}
	if c.userID > 0 {
		connPayload.UserID = c.userID
	}
	c.sendJSON(Envelope{
		Type:     ServerConnected,
		ClientID: c.clientID,
		RoomID:   roomID,
		Seq:      currentSeq,
		Payload:  marshalPayload(connPayload),
	})

	// 首次连接也推送当前快照（lastSeq=0）
	if payload.LastSeq == 0 {
		c.sendJSON(Envelope{
			Type:     ServerSync,
			ClientID: c.clientID,
			RoomID:   roomID,
			Payload:  marshalPayload(SyncPayload{Snapshot: snap, Events: nil}),
		})
	}
}

func roomEventsFromStored(rawList [][]byte) []RoomEvent {
	out := make([]RoomEvent, 0, len(rawList))
	for _, raw := range rawList {
		if ev := parseRoomEventFromEnvelope(raw); ev != nil {
			out = append(out, *ev)
		}
	}
	return out
}

func (h *Hub) handleClientMessage(c *Client, data []byte) {
	var env Envelope
	if err := json.Unmarshal(data, &env); err != nil {
		return
	}

	switch env.Type {
	case ClientSubscribe:
		var p SubscribePayload
		if len(env.Payload) > 0 {
			_ = json.Unmarshal(env.Payload, &p)
		}
		if p.ClientID == "" {
			p.ClientID = env.ClientID
		}
		if p.LastSeq == 0 {
			p.LastSeq = env.LastSeq
		}
		h.handleSubscribe(c, p)
	case ClientPing:
		var p PingPayload
		if len(env.Payload) > 0 {
			_ = json.Unmarshal(env.Payload, &p)
		}
		if p.LastSeq > 0 {
			c.setLastSeq(p.LastSeq)
		}
		c.sendJSON(Envelope{
			Type:      ServerPong,
			ClientID:  c.clientID,
			RoomID:    c.roomID,
			Timestamp: time.Now().UnixMilli(),
			Payload:   marshalPayload(map[string]uint64{"lastSeq": c.getLastSeq()}),
		})
	case ClientBid:
		h.handleBid(c, env.Payload)
	}
}

func (h *Hub) handleBid(c *Client, raw json.RawMessage) {
	if c.userID == 0 {
		c.sendJSON(Envelope{
			Type:    ServerError,
			Payload: marshalPayload(ErrorPayload{Code: 401, Message: "login required for bid"}),
		})
		return
	}
	if !h.limiter.allow(c.userID) {
		c.sendJSON(Envelope{
			Type:    ServerError,
			Payload: marshalPayload(ErrorPayload{Code: 429, Message: "bid too frequent"}),
		})
		return
	}

	var body BidPayload
	if err := json.Unmarshal(raw, &body); err != nil {
		return
	}

	ctx := context.Background()
	session, err := h.sessions.GetByRoomID(ctx, c.roomID)
	if err != nil {
		c.sendJSON(Envelope{
			Type:    ServerError,
			Payload: marshalPayload(ErrorPayload{Code: 404, Message: "session not found"}),
		})
		return
	}

	metrics.RecordBidAttempt()
	result, err := h.bidSvc.PlaceBid(ctx, c.userID, session.ID, service.PlaceBidInput{
		Amount:    body.Amount,
		RequestID: body.RequestID,
	})
	if err != nil {
		metrics.RecordBidFailure()
		c.sendJSON(Envelope{
			Type:    ServerError,
			Payload: marshalPayload(ErrorPayload{Code: 400, Message: err.Error()}),
		})
		return
	}
	metrics.RecordBidSuccess()

	// PlaceBid 内 Notifier 会广播；此处仅回 ACK
	_ = result
}

// ClientCount 返回指定房间的 WS 连接数（用于主播端进房人数统计）
func (h *Hub) ClientCount(roomID string) int {
	h.mu.RLock()
	r, ok := h.rooms[roomID]
	h.mu.RUnlock()
	if !ok {
		return 0
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients)
}

// Stats 返回当前 WS 连接数与活跃房间数（5.6）
func (h *Hub) Stats() (connections int, rooms int) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	rooms = len(h.rooms)
	for _, r := range h.rooms {
		r.mu.RLock()
		connections += len(r.clients)
		r.mu.RUnlock()
	}
	return connections, rooms
}

func (h *Hub) runCountdownLoop() {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		h.mu.RLock()
		roomIDs := make([]string, 0, len(h.rooms))
		for id, r := range h.rooms {
			r.mu.RLock()
			hasClients := len(r.clients) > 0
			r.mu.RUnlock()
			if hasClients {
				roomIDs = append(roomIDs, id)
			}
		}
		h.mu.RUnlock()

		for _, roomID := range roomIDs {
			ctx := context.Background()
			snap, err := h.snap.SnapshotByRoom(ctx, roomID)
			if err != nil || snap.Status != domain.SessionStatusRunning {
				continue
			}
			h.broadcastEphemeral(roomID, EventCountdownTick, snap)
		}
	}
}
