// Package auth provides authentication services for the application.
package auth

import (
	"errors"
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/crypt"
	"strconv"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AuthService struct {
	userRepo *repo.UserRepository

	jwtService  *crypt.JwtService
	jwtDuration time.Duration
	logger      *zap.Logger
}

func NewAuthService(userRepo *repo.UserRepository, jwtService *crypt.JwtService, jwtDuration time.Duration, logger *zap.Logger) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		logger:      logger,
		jwtService:  jwtService,
		jwtDuration: jwtDuration,
	}
}

func (s *AuthService) Authenticate(uID int64) (string, error) {
	_, err := s.userRepo.FindByID(uID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Info("failed to find user by UID, register", zap.Int64("UID", uID))
			_, err := s.userRepo.Create(uID)
			if err != nil {
				s.logger.Info("failed to register user", zap.Int64("UID", uID), zap.Error(err))
				return "", err
			}
			s.logger.Info("user registered successfully", zap.Int64("UID", uID))
		} else {
			s.logger.Info("failed to find user by UID", zap.Int64("UID", uID), zap.Error(err))
			return "", err
		}
	}

	token, err := s.jwtService.GenerateToken(strconv.Itoa(int(uID)), s.jwtDuration)
	s.logger.Debug("user authenticated", zap.Int64("user_id", uID), zap.Int64("UID", uID))
	return token, nil
}

func (s *AuthService) GenerateToken(userID int64) (string, error) {
	token, err := s.jwtService.GenerateToken(strconv.Itoa(int(userID)), s.jwtDuration)
	if err != nil {
		s.logger.Error("failed to generate refresh token", zap.Int64("user_id", userID), zap.Error(err))
		return "", err
	}
	return token, nil
}
