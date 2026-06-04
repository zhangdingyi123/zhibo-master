package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/zhibo/backend/internal/auth"
	"github.com/zhibo/backend/internal/domain"
	"github.com/zhibo/backend/internal/repository"
)

var phoneRe = regexp.MustCompile(`^1[3-9]\d{9}$`)

type AuthService struct {
	users  *repository.UserRepo
	secret string
}

func NewAuthService(users *repository.UserRepo, jwtSecret string) *AuthService {
	return &AuthService{users: users, secret: jwtSecret}
}

type RegisterInput struct {
	Phone    string
	Password string
	Nickname string
	Role     domain.UserRole
}

type AuthResult struct {
	Token string       `json:"token"`
	User  *domain.User `json:"user"`
}

func (s *AuthService) Register(ctx context.Context, in RegisterInput) (*AuthResult, error) {
	phone := normalizePhone(in.Phone)
	if !phoneRe.MatchString(phone) {
		return nil, domain.ErrInvalidPhone
	}
	if len(in.Password) < 6 {
		return nil, domain.ErrWeakPassword
	}
	nickname := strings.TrimSpace(in.Nickname)
	if nickname == "" {
		return nil, domain.ErrInvalidNickname
	}
	role := in.Role
	if role == "" {
		role = domain.UserRoleBuyer
	}
	if role != domain.UserRoleBuyer && role != domain.UserRoleAnchor {
		return nil, domain.ErrRoleNotAllowed
	}

	if _, err := s.users.GetByPhone(ctx, phone); err == nil {
		return nil, domain.ErrPhoneAlreadyExists
	} else if err != domain.ErrNotFound {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	openID := fmt.Sprintf("u_%s", phone)
	u := &domain.User{
		OpenID:   openID,
		Phone:    phone,
		Nickname: nickname,
		Avatar:   defaultAvatar(phone),
		Role:     role,
	}
	if err := s.users.Create(ctx, u, string(hash)); err != nil {
		return nil, err
	}
	return s.tokenFor(u)
}

func (s *AuthService) Login(ctx context.Context, phone, password string) (*AuthResult, error) {
	phone = normalizePhone(phone)
	u, hash, err := s.users.GetByPhoneWithHash(ctx, phone)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, domain.ErrInvalidCredentials
		}
		return nil, err
	}
	if hash == "" || bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		return nil, domain.ErrInvalidCredentials
	}
	return s.tokenFor(u)
}

func (s *AuthService) Me(ctx context.Context, userID uint64) (*domain.User, error) {
	return s.users.GetByID(ctx, userID)
}

func (s *AuthService) tokenFor(u *domain.User) (*AuthResult, error) {
	token, err := auth.IssueToken(s.secret, u)
	if err != nil {
		return nil, fmt.Errorf("issue token: %w", err)
	}
	return &AuthResult{Token: token, User: u}, nil
}

func normalizePhone(p string) string {
	return strings.TrimSpace(p)
}

func defaultAvatar(phone string) string {
	return fmt.Sprintf("https://picsum.photos/seed/%s/200", phone)
}
