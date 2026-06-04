package ws

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/zhibo/backend/internal/domain"
	"github.com/zhibo/backend/internal/infra/metrics"
	"github.com/zhibo/backend/internal/repository"
	"github.com/zhibo/backend/internal/service"
)

// Hub 按 roomId 隔离的 WebSocket 房间网关（4.1）
type Hub struct {
	rooms    map[string]*room
	mu       sync.RWMutex
	sessions *repository.SessionRepo
	bids     *repository.BidRepo
	snap     *service.UserAuctionService
	bidSvc   *service.BidService
	limiter  *bidRateLimiter

	registerCh   chan *Client
	unregisterCh chan *Client
	broadcast    chan roomBroadcast
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

// NewHub 创建 WS 网关
func NewHub(
	sessions *repository.SessionRepo,
	bids *repository.BidRepo,
	snap *service.UserAuctionService,
	bidSvc *service.BidService,
) *Hub {
	h := &Hub{
		rooms:      make(map[string]*room),
		sessions:   sessions,
		bids:       bids,
		snap:       snap,
		bidSvc:     bidSvc,
		limiter:    newBidRateLimiter(300 * time.Millisecond),
		registerCh:   make(chan *Client),
		unregisterCh: make(chan *Client),
		broadcast:    make(chan roomBroadcast, 256),
	}
	go h.run()
	go h.runCountdownLoop()
	return h
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
	r := h.getOrCreateRoom(roomID)
	ev := RoomEvent{
		Type:    eventType,
		Ts:      time.Now().UnixMilli(),
		Payload: marshalPayload(payload),
	}
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

	r := h.getOrCreateRoom(roomID)
	currentSeq := r.store.CurrentSeq()

	snap, err := h.snap.SnapshotByRoom(ctx, roomID)
	if err != nil {
		c.sendJSON(Envelope{
			Type:    ServerError,
			Payload: marshalPayload(ErrorPayload{Code: 404, Message: err.Error()}),
		})
		return
	}

	// 重连补偿（4.6）
	if payload.LastSeq > 0 && payload.LastSeq < currentSeq {
		missed := r.store.Since(payload.LastSeq)
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

// runCountdownLoop 服务端权威倒计时广播（4.5），200ms 精度
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
