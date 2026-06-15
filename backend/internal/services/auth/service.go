package auth

import (
	"errors"
	"strings"
	"time"

	"bitapi/backend/internal/config"
	"bitapi/backend/internal/models"
	bcrypto "bitapi/backend/internal/pkg/crypto"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials = errors.New("账号或密码错误")
	ErrTokenInvalid       = errors.New("令牌无效或已过期")
)

type Service struct {
	db  *gorm.DB
	cfg config.Config
}

type TokenPair struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	ExpiresAt    time.Time   `json:"expires_at"`
	User         models.User `json:"user"`
}

type Claims struct {
	UserID       uint   `json:"uid"`
	Role         string `json:"role"`
	TokenVersion int    `json:"tver"`
	jwt.RegisteredClaims
}

func New(db *gorm.DB, cfg config.Config) *Service {
	return &Service{db: db, cfg: cfg}
}

func (s *Service) Register(email, password, displayName string) (TokenPair, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if displayName == "" {
		displayName = email
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return TokenPair{}, err
	}
	user := models.User{
		Email:         email,
		PasswordHash:  string(hash),
		DisplayName:   displayName,
		Role:          models.RoleUser,
		Status:        models.StatusActive,
		TokenVersion:  1,
		BalanceMicros: s.cfg.DefaultUserBalance,
	}
	if err := s.db.Create(&user).Error; err != nil {
		return TokenPair{}, err
	}
	return s.issuePair(user)
}

func (s *Service) Login(email, password string) (TokenPair, error) {
	var user models.User
	err := s.db.Where("email = ?", strings.ToLower(strings.TrimSpace(email))).First(&user).Error
	if err != nil {
		return TokenPair{}, ErrInvalidCredentials
	}
	if user.Status != models.StatusActive {
		return TokenPair{}, ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return TokenPair{}, ErrInvalidCredentials
	}
	now := time.Now()
	user.LastLoginAt = &now
	user.LastActiveAt = &now
	_ = s.db.Save(&user).Error
	return s.issuePair(user)
}

func (s *Service) Refresh(refreshToken string) (TokenPair, error) {
	hash := bcrypto.SHA256Hex(refreshToken)
	var stored models.RefreshToken
	if err := s.db.Where("token_hash = ?", hash).First(&stored).Error; err != nil {
		return TokenPair{}, ErrTokenInvalid
	}
	if stored.RevokedAt != nil || time.Now().After(stored.ExpiresAt) {
		return TokenPair{}, ErrTokenInvalid
	}
	var user models.User
	if err := s.db.First(&user, stored.UserID).Error; err != nil {
		return TokenPair{}, ErrTokenInvalid
	}
	if user.TokenVersion != stored.TokenVersion || user.Status != models.StatusActive {
		return TokenPair{}, ErrTokenInvalid
	}
	now := time.Now()
	stored.RevokedAt = &now
	_ = s.db.Save(&stored).Error
	return s.issuePair(user)
}

func (s *Service) ParseAccessToken(tokenValue string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenValue, &Claims{}, func(token *jwt.Token) (any, error) {
		return []byte(s.cfg.JWTSecret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrTokenInvalid
	}
	var user models.User
	if err := s.db.Select("id", "token_version", "status").First(&user, claims.UserID).Error; err != nil {
		return nil, err
	}
	if user.TokenVersion != claims.TokenVersion || user.Status != models.StatusActive {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}

func (s *Service) issuePair(user models.User) (TokenPair, error) {
	expiresAt := time.Now().Add(s.cfg.AccessTokenTTL)
	claims := Claims{
		UserID:       user.ID,
		Role:         user.Role,
		TokenVersion: user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uuid.NewString(),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    s.cfg.AppName,
		},
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(s.cfg.JWTSecret))
	if err != nil {
		return TokenPair{}, err
	}
	refreshToken, err := bcrypto.RandomToken("bar", 48)
	if err != nil {
		return TokenPair{}, err
	}
	stored := models.RefreshToken{
		UserID:       user.ID,
		FamilyID:     uuid.NewString(),
		TokenHash:    bcrypto.SHA256Hex(refreshToken),
		TokenVersion: user.TokenVersion,
		ExpiresAt:    time.Now().Add(s.cfg.RefreshTokenTTL),
	}
	if err := s.db.Create(&stored).Error; err != nil {
		return TokenPair{}, err
	}
	return TokenPair{AccessToken: accessToken, RefreshToken: refreshToken, ExpiresAt: expiresAt, User: user}, nil
}
