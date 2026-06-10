package service

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/zhibo/backend/internal/domain"
	"github.com/zhibo/backend/internal/repository"
)

const (
	SocialEventCommentNew   = "comment.new"
	SocialEventLikeUpdate   = "like.update"
	SocialEventCommentHidden = "comment.hidden"
)

type EventBroadcaster interface {
	Publish(roomID, eventType string, payload any) uint64
}

type SocialService struct {
	social    *repository.SocialRepo
	users     *repository.UserRepo
	liveRooms *repository.LiveRoomRepo
	sessions  *repository.SessionRepo
	products  *repository.ProductRepo
	broadcast EventBroadcaster
}

func NewSocialService(
	social *repository.SocialRepo,
	users *repository.UserRepo,
	liveRooms *repository.LiveRoomRepo,
	sessions *repository.SessionRepo,
	products *repository.ProductRepo,
) *SocialService {
	return &SocialService{
		social:    social,
		users:     users,
		liveRooms: liveRooms,
		sessions:  sessions,
		products:  products,
	}
}

func (s *SocialService) SetBroadcaster(b EventBroadcaster) { s.broadcast = b }

type RoomSocialStats struct {
	LikeCount    int  `json:"likeCount"`
	CommentCount int  `json:"commentCount"`
	IsFollowing  bool `json:"isFollowing,omitempty"`
	IsFavorited  bool `json:"isFavorited,omitempty"`
}

type AnchorBrief struct {
	ID            uint64 `json:"id"`
	Nickname      string `json:"nickname"`
	Avatar        string `json:"avatar"`
	FollowerCount int    `json:"followerCount"`
}

type CommentNewPayload struct {
	Comment domain.RoomComment `json:"comment"`
}

type LikeUpdatePayload struct {
	LikeCount int `json:"likeCount"`
}

func (s *SocialService) GetRoomStats(ctx context.Context, roomID string, viewerID *uint64, productID *uint64) (*RoomSocialStats, error) {
	likes, err := s.social.CountLikes(ctx, roomID)
	if err != nil {
		return nil, err
	}
	comments, err := s.social.CountComments(ctx, roomID, true)
	if err != nil {
		return nil, err
	}
	out := &RoomSocialStats{LikeCount: likes, CommentCount: comments}
	if viewerID != nil {
		if anchorID, err := s.roomAnchorID(ctx, roomID); err == nil {
			following, _ := s.social.IsFollowing(ctx, *viewerID, anchorID)
			out.IsFollowing = following
		}
		if productID != nil {
			fav, _ := s.social.IsFavorite(ctx, *viewerID, *productID)
			out.IsFavorited = fav
		}
	}
	return out, nil
}

func (s *SocialService) GetAnchorBrief(ctx context.Context, anchorID uint64) (*AnchorBrief, error) {
	u, err := s.users.GetByID(ctx, anchorID)
	if err != nil {
		return nil, err
	}
	followers, _ := s.social.CountFollowers(ctx, anchorID)
	return &AnchorBrief{
		ID:            u.ID,
		Nickname:      u.Nickname,
		Avatar:        u.Avatar,
		FollowerCount: followers,
	}, nil
}

func (s *SocialService) ListComments(ctx context.Context, roomID string, includeHidden bool) ([]domain.RoomComment, error) {
	return s.social.ListComments(ctx, roomID, includeHidden, 50)
}

func (s *SocialService) PostComment(ctx context.Context, userID uint64, roomID, content string) (*domain.RoomComment, error) {
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, domain.ErrCommentEmpty
	}
	if utf8.RuneCountInString(content) > 200 {
		return nil, domain.ErrCommentTooLong
	}
	if _, err := s.roomAnchorID(ctx, roomID); err != nil {
		return nil, domain.ErrNotFound
	}
	id, err := s.social.InsertComment(ctx, roomID, userID, content)
	if err != nil {
		return nil, err
	}
	c, err := s.social.GetComment(ctx, id)
	if err != nil {
		return nil, err
	}
	if s.broadcast != nil {
		s.broadcast.Publish(roomID, SocialEventCommentNew, CommentNewPayload{Comment: *c})
	}
	return c, nil
}

func (s *SocialService) HideComment(ctx context.Context, anchorID uint64, commentID uint64) error {
	c, err := s.social.GetComment(ctx, commentID)
	if err != nil {
		return err
	}
	if err := s.assertRoomAnchor(ctx, c.RoomID, anchorID); err != nil {
		return err
	}
	if err := s.social.HideComment(ctx, commentID); err != nil {
		return err
	}
	if s.broadcast != nil {
		s.broadcast.Publish(c.RoomID, SocialEventCommentHidden, map[string]any{"commentId": commentID})
	}
	return nil
}

func (s *SocialService) Like(ctx context.Context, userID uint64, roomID string) (int, error) {
	if _, err := s.roomAnchorID(ctx, roomID); err != nil {
		return 0, domain.ErrNotFound
	}
	_ = s.social.InsertLike(ctx, roomID, userID)
	count, err := s.social.CountLikes(ctx, roomID)
	if err != nil {
		return 0, err
	}
	if s.broadcast != nil {
		s.broadcast.Publish(roomID, SocialEventLikeUpdate, LikeUpdatePayload{LikeCount: count})
	}
	return count, nil
}

func (s *SocialService) ToggleFavorite(ctx context.Context, userID, productID uint64) (bool, error) {
	p, err := s.products.GetByID(ctx, productID)
	if err != nil {
		return false, err
	}
	_ = p
	return s.social.ToggleFavorite(ctx, userID, productID)
}

func (s *SocialService) ToggleFollow(ctx context.Context, userID, anchorID uint64) (bool, error) {
	if userID == anchorID {
		return false, domain.ErrCannotFollowSelf
	}
	u, err := s.users.GetByID(ctx, anchorID)
	if err != nil {
		return false, err
	}
	if u.Role != domain.UserRoleAnchor && u.Role != domain.UserRoleAdmin {
		return false, domain.ErrNotFound
	}
	return s.social.ToggleFollow(ctx, userID, anchorID)
}

func (s *SocialService) assertRoomAnchor(ctx context.Context, roomID string, anchorID uint64) error {
	owner, err := s.roomAnchorID(ctx, roomID)
	if err != nil {
		return domain.ErrForbidden
	}
	if owner != anchorID {
		return domain.ErrForbidden
	}
	return nil
}

func (s *SocialService) roomAnchorID(ctx context.Context, roomID string) (uint64, error) {
	lr, err := s.liveRooms.GetByRoomID(ctx, roomID)
	if err == nil {
		return lr.AnchorID, nil
	}
	sess, err := s.sessions.GetByRoomID(ctx, roomID)
	if err != nil {
		return 0, err
	}
	return sess.AnchorID, nil
}
