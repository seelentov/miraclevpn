package crypt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

var (
	ErrInvalidToken = errors.New("invalid token")
)

type JwtService struct {
	secretKey string
	logger    *zap.Logger
}

func NewJwtService(secret string, logger *zap.Logger) *JwtService {
	return &JwtService{secretKey: secret, logger: logger}
}

type Claims struct {
	UserID string `json:"user_id"`
	jwt.RegisteredClaims
}

func (j *JwtService) GenerateToken(userID string, duration time.Duration) (string, error) {
	j.logger.Debug("generating JWT token", zap.String("user_id", userID), zap.Duration("duration", duration))
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt: jwt.NewNumericDate(time.Now()),
		},
	}
	if duration > 0 {
		claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(duration))
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(j.secretKey))
	if err != nil {
		j.logger.Error("failed to sign JWT token", zap.String("user_id", userID), zap.Error(err))
		return "", err
	}
	j.logger.Debug("JWT token generated", zap.String("user_id", userID))
	return signed, nil
}

func (j *JwtService) ParseToken(tokenStr string) (*Claims, error) {
	j.logger.Debug("parsing JWT token")
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(j.secretKey), nil
	})
	if err != nil {
		j.logger.Warn("failed to parse JWT token", zap.Error(err))
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		j.logger.Warn("invalid JWT token")
		return nil, ErrInvalidToken
	}
	j.logger.Debug("JWT token parsed", zap.String("user_id", claims.UserID))
	return claims, nil
}
