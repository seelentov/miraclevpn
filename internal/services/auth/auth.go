// Package auth provides authentication services for the application.
package auth

import (
	"errors"
	"fmt"
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/crypt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrBanned    = errors.New("user is banned")
	ErrExpired   = errors.New("user is expired")
	ErrNewDevice = errors.New("user logged by new device")
)

type AuthService struct {
	userRepo     *repo.UserRepository
	authDataRepo *repo.AuthDataRepository

	jwtService  *crypt.JwtService
	jwtDuration time.Duration
	logger      *zap.Logger
}

func NewAuthService(userRepo *repo.UserRepository, authDataRepo *repo.AuthDataRepository, jwtService *crypt.JwtService, jwtDuration time.Duration, logger *zap.Logger) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		logger:       logger,
		jwtService:   jwtService,
		jwtDuration:  jwtDuration,
		authDataRepo: authDataRepo,
	}
}

func (s *AuthService) Authenticate(uID string, data map[string]interface{}, saveAuthData bool) (string, error) {
	_, err := s.userRepo.FindByID(uID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Info("failed to find user by UID, register", zap.String("UID", uID))
			_, err := s.userRepo.Create(uID)
			if err != nil {
				s.logger.Info("failed to register user", zap.String("UID", uID), zap.Error(err))
				return "", err
			}
			s.logger.Info("user registered successfully", zap.String("UID", uID))
		} else {
			s.logger.Info("failed to find user by UID", zap.String("UID", uID), zap.Error(err))
			return "", err
		}
	}

	if saveAuthData {
		latestAuth, err := s.authDataRepo.FindLatest(uID)
		if err != nil {
			return "", err
		}

		if latestAuth != nil {
			latestData := latestAuth.Data

			fields := []string{
				"brand",
				"designName",
				"manufacturer",
				"modelName",
				"deviceYearClass",
				"osName",
				"productName",
			}

			for _, field := range fields {
				if data[field] != latestData[field] {
					return "", fmt.Errorf("%w: %s - %v -> %v", ErrNewDevice, field, latestData, data)
				}
			}

		}

		if err := s.authDataRepo.Add(
			uID,
			data,
		); err != nil {
			return "", err
		}
	}

	token, err := s.jwtService.GenerateToken(map[string]string{
		"user_id": uID,
	}, s.jwtDuration)
	if err != nil {
		return "", err
	}
	s.logger.Debug("user authenticated", zap.String("user_id", uID), zap.String("UID", uID))
	return token, nil
}

func (s *AuthService) GenerateToken(userID string) (string, error) {
	token, err := s.jwtService.GenerateToken(map[string]string{
		"user_id": userID,
	}, s.jwtDuration)
	if err != nil {
		s.logger.Error("failed to generate refresh token", zap.String("user_id", userID), zap.Error(err))
		return "", err
	}
	return token, nil
}
