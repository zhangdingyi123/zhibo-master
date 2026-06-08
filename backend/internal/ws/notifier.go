package ws

import (
	"context"
	"time"

	"github.com/zhibo/backend/internal/domain"
	redisc "github.com/zhibo/backend/internal/infra/redis"
	"github.com/zhibo/backend/internal/repository"
	"github.com/zhibo/backend/internal/service"
)

// Notifier 将业务事件推送到 WebSocket 房间
type Notifier struct {
	hub   *Hub
	bids  *repository.BidRepo
	cache service.RoomCache
}

func NewNotifier(hub *Hub, bids *repository.BidRepo) *Notifier {
	return &Notifier{hub: hub, bids: bids}
}

// SetRoomCache 排行榜读缓存（5.1）
func (n *Notifier) SetRoomCache(c service.RoomCache) {
	n.cache = c
}

// BidNewPayload 新出价事件
type BidNewPayload struct {
	Bid      domain.Bid              `json:"bid"`
	Snapshot *service.SessionSnapshot `json:"snapshot"`
}

// RankEntry 排行榜项
type RankEntry struct {
	UserID   uint64 `json:"userId"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Amount   int64  `json:"amount"`
	Seq      uint32 `json:"seq"`
	Rank     int    `json:"rank"`
}

// RankUpdatePayload 排名更新
type RankUpdatePayload struct {
	Items []RankEntry `json:"items"`
}

// ExtendedPayload 延时
type ExtendedPayload struct {
	Snapshot   *service.SessionSnapshot `json:"snapshot"`
	PreviousEndAtMs int64               `json:"previousEndAtMs"`
	NewEndAtMs      int64               `json:"newEndAtMs"`
}

// SettledPayload 成交
type SettledPayload struct {
	Session  domain.AuctionSession  `json:"session"`
	Snapshot *service.SessionSnapshot `json:"snapshot"`
	Order    *domain.Order          `json:"order,omitempty"`
}

// CancelledPayload 取消
type CancelledPayload struct {
	Session  domain.AuctionSession  `json:"session"`
	Snapshot *service.SessionSnapshot `json:"snapshot"`
	Reason   string                 `json:"reason"`
}

func (n *Notifier) OnBid(ctx context.Context, result *service.PlaceBidResult, prevEndAt *time.Time) {
	if n == nil || n.hub == nil || result == nil {
		return
	}
	roomID := result.Session.RoomID
	snap := result.Snapshot

	n.hub.Publish(roomID, EventBidNew, BidNewPayload{
		Bid:      result.Bid,
		Snapshot: snap,
	})

	items := n.loadRankTop(ctx, roomID, result.Session.ID)
	if len(items) > 0 {
		n.hub.Publish(roomID, EventRankUpdate, RankUpdatePayload{Items: items})
	}

	if prevEndAt != nil && snap.EndAtMs != nil && result.Session.Status == domain.SessionStatusRunning {
		newMs := *snap.EndAtMs
		prevMs := prevEndAt.UnixMilli()
		if newMs > prevMs {
			n.hub.Publish(roomID, EventAuctionExtended, ExtendedPayload{
				Snapshot:        snap,
				PreviousEndAtMs: prevMs,
				NewEndAtMs:      newMs,
			})
		}
	}

	if result.Settled {
		n.OnSettled(ctx, &result.Session, result.Order)
	}
}

func (n *Notifier) OnSettled(ctx context.Context, session *domain.AuctionSession, order *domain.Order) {
	if n == nil || n.hub == nil || session == nil {
		return
	}
	snap := service.BuildSnapshot(session, time.Now())
	n.hub.Publish(session.RoomID, EventAuctionSettled, SettledPayload{
		Session:  *session,
		Snapshot: snap,
		Order:    order,
	})
}

func (n *Notifier) OnCancelled(ctx context.Context, session *domain.AuctionSession, reason string) {
	if n == nil || n.hub == nil || session == nil {
		return
	}
	snap := service.BuildSnapshot(session, time.Now())
	n.hub.Publish(session.RoomID, EventAuctionCancelled, CancelledPayload{
		Session:  *session,
		Snapshot: snap,
		Reason:   reason,
	})
}

// BroadcastCountdown 非持久化倒计时推送（不计入 seq）
func (n *Notifier) loadRankTop(ctx context.Context, roomID string, sessionID uint64) []RankEntry {
	if n.cache != nil {
		if cached, err := n.cache.GetRankTop(ctx, roomID); err == nil && len(cached) > 0 {
			items := make([]RankEntry, len(cached))
			for i, m := range cached {
				items[i] = RankEntry{
					UserID:   m.UserID,
					Nickname: m.Nickname,
					Avatar:   m.Avatar,
					Amount:   m.Amount,
					Seq:      m.Seq,
					Rank:     m.Rank,
				}
			}
			return items
		}
	}
	rows, err := n.bids.ListTopBySession(ctx, sessionID, 10)
	if err != nil || len(rows) == 0 {
		return nil
	}
	items := make([]RankEntry, len(rows))
	members := make([]redisc.RankMember, len(rows))
	for i, row := range rows {
		items[i] = RankEntry{
			UserID:   row.UserID,
			Nickname: row.Nickname,
			Avatar:   row.Avatar,
			Amount:   row.Amount,
			Seq:      row.Seq,
			Rank:     row.Rank,
		}
		members[i] = redisc.RankMember{
			UserID:   row.UserID,
			Nickname: row.Nickname,
			Avatar:   row.Avatar,
			Amount:   row.Amount,
			Seq:      row.Seq,
			Rank:     row.Rank,
		}
	}
	if n.cache != nil {
		_ = n.cache.SetRankTop(ctx, roomID, members)
	}
	return items
}

func (n *Notifier) BroadcastCountdown(roomID string, snap *service.SessionSnapshot) {
	if n == nil || n.hub == nil {
		return
	}
	n.hub.broadcastEphemeral(roomID, EventCountdownTick, snap)
}
