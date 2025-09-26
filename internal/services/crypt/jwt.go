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
	Data map[string]string `json:"data"`
	jwt.RegisteredClaims
}

func (j *JwtService) GenerateToken(data map[string]string, duration time.Duration) (string, error) {
	claims := &Claims{
		Data: data,
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
		j.logger.Error("failed to sign JWT token", zap.Error(err))
		return "", err
	}
	return signed, nil
}

func (j *JwtService) ParseToken(tokenStr string) (*Claims, error) {
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
	return claims, nil
}
