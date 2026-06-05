package service

import (
	"context"
	"fmt"
	"time"

	"github.com/zhibo/backend/internal/domain"
	"github.com/zhibo/backend/internal/repository"
)

// MessageService 写扩散消息：事件发生时写入各用户收件箱
type MessageService struct {
	messages *repository.MessageRepo
	bids     *repository.BidRepo
}

func NewMessageService(messages *repository.MessageRepo, bids *repository.BidRepo) *MessageService {
	return &MessageService{messages: messages, bids: bids}
}

type ListMessagesInput struct {
	UnreadOnly bool
	Page       int
	PageSize   int
}

type ListMessagesResult struct {
	Items    []domain.UserMessage `json:"items"`
	Total    int                  `json:"total"`
	Unread   int                  `json:"unread"`
	Page     int                  `json:"page"`
	PageSize int                  `json:"pageSize"`
}

func (s *MessageService) List(ctx context.Context, userID uint64, in ListMessagesInput) (*ListMessagesResult, error) {
	items, total, err := s.messages.ListByUser(ctx, userID, in.UnreadOnly, in.Page, in.PageSize)
	if err != nil {
		return nil, err
	}
	unread, err := s.messages.CountUnread(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &ListMessagesResult{
		Items:    items,
		Total:    total,
		Unread:   unread,
		Page:     in.Page,
		PageSize: in.PageSize,
	}, nil
}

func (s *MessageService) UnreadCount(ctx context.Context, userID uint64) (int, error) {
	return s.messages.CountUnread(ctx, userID)
}

func (s *MessageService) MarkRead(ctx context.Context, userID, messageID uint64) error {
	return s.messages.MarkRead(ctx, userID, messageID)
}

func (s *MessageService) MarkAllRead(ctx context.Context, userID uint64) (int64, error) {
	return s.messages.MarkAllRead(ctx, userID)
}

// FanOutOnBid 出价后写扩散
func (s *MessageService) FanOutOnBid(ctx context.Context, result *PlaceBidResult, prevEndAt *time.Time, prevWinnerID *uint64) {
	if s == nil || result == nil {
		return
	}
	session := result.Session
	bidder := result.Bid.UserID
	payload := map[string]any{
		"sessionId": session.ID,
		"roomId":    session.RoomID,
		"bidSeq":    result.Bid.Seq,
		"amount":    result.Bid.Amount,
	}

	if prevWinnerID != nil && *prevWinnerID != bidder {
		_ = s.messages.Insert(ctx, repository.InsertMessageInput{
			UserID:    *prevWinnerID,
			EventType: domain.MessageOutbid,
			Category:  domain.MessageCategoryAuction,
			Title:     "领先位被抢",
			Body:      "快出一手夺回领先！",
			Payload:   payload,
			DedupeKey: fmt.Sprintf("outbid:%d:%d", session.ID, result.Bid.Seq),
		})
	}

	if prevEndAt != nil && session.EndAt != nil && session.Status == domain.SessionStatusRunning {
		newMs := session.EndAt.UnixMilli()
		prevMs := prevEndAt.UnixMilli()
		if newMs > prevMs {
			s.fanOutExcept(ctx, session.ID, bidder, domain.MessageExtended, "竞拍延时",
				"结束前有人出价，倒计时已延长",
				fmt.Sprintf("extended:%d:%d", session.ID, result.Bid.Seq), payload)
		}
	}

	if result.Settled {
		s.fanOutSettled(ctx, session, bidder, result.Order, payload)
	}
}

// FanOutOnCancelled 取消场次写扩散
func (s *MessageService) FanOutOnCancelled(ctx context.Context, session *domain.AuctionSession, reason string) {
	if s == nil || session == nil {
		return
	}
	payload := map[string]any{
		"sessionId": session.ID,
		"roomId":    session.RoomID,
		"reason":    reason,
	}
	body := reason
	if body == "" {
		body = "主播已取消本场竞拍"
	}
	participants, err := s.bids.ListParticipantUserIDs(ctx, session.ID)
	if err != nil {
		return
	}
	for _, uid := range participants {
		_ = s.messages.Insert(ctx, repository.InsertMessageInput{
			UserID:    uid,
			EventType: domain.MessageCancelled,
			Category:  domain.MessageCategoryAuction,
			Title:     "竞拍已取消",
			Body:      body,
			Payload:   payload,
			DedupeKey: fmt.Sprintf("cancelled:%d", session.ID),
		})
	}
}

func (s *MessageService) fanOutSettled(ctx context.Context, session domain.AuctionSession, winnerID uint64, order *domain.Order, base map[string]any) {
	participants, err := s.bids.ListParticipantUserIDs(ctx, session.ID)
	if err != nil {
		return
	}
	payload := make(map[string]any, len(base)+2)
	for k, v := range base {
		payload[k] = v
	}
	if order != nil {
		payload["orderId"] = order.ID
	}

	for _, uid := range participants {
		if uid == winnerID {
			_ = s.messages.Insert(ctx, repository.InsertMessageInput{
				UserID:    uid,
				EventType: domain.MessageSettledWin,
				Category:  domain.MessageCategoryAuction,
				Title:     "恭喜中标",
				Body:      "本场竞拍您已胜出，请尽快完成支付",
				Payload:   payload,
				DedupeKey: fmt.Sprintf("settled_win:%d", session.ID),
			})
		} else {
			_ = s.messages.Insert(ctx, repository.InsertMessageInput{
				UserID:    uid,
				EventType: domain.MessageSettled,
				Category:  domain.MessageCategoryAuction,
				Title:     "竞拍结束",
				Body:      "本场竞拍已成交",
				Payload:   payload,
				DedupeKey: fmt.Sprintf("settled:%d:%d", session.ID, uid),
			})
		}
	}
}

func (s *MessageService) fanOutExcept(ctx context.Context, sessionID, except uint64, event domain.MessageEventType, title, body, dedupe string, payload map[string]any) {
	participants, err := s.bids.ListParticipantUserIDs(ctx, sessionID)
	if err != nil {
		return
	}
	for _, uid := range participants {
		if uid == except {
			continue
		}
		_ = s.messages.Insert(ctx, repository.InsertMessageInput{
			UserID:    uid,
			EventType: event,
			Category:  domain.MessageCategoryAuction,
			Title:     title,
			Body:      body,
			Payload:   payload,
			DedupeKey: dedupe,
		})
	}
}
