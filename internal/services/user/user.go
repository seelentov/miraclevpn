// Package user provides user management services for the application.
package user

import (
	"errors"
	"miraclevpn/internal/models"
	"miraclevpn/internal/repo"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrNotFound = errors.New("user not found")
)

type UserService struct {
	userRepo *repo.UserRepository
	logger   *zap.Logger
}

func NewUserService(userRepo *repo.UserRepository, logger *zap.Logger) *UserService {
	return &UserService{
		userRepo: userRepo,
		logger:   logger,
	}
}

func (s *UserService) GetUserByID(id int64) (*models.User, error) {
	s.logger.Debug("getting user by id", zap.Int64("user_id", id))
	u, err := s.userRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Warn("user not found", zap.Int64("user_id", id))
			return nil, ErrNotFound
		}
		s.logger.Error("failed to get user by id", zap.Int64("user_id", id), zap.Error(err))
		return nil, err
	}
	s.logger.Debug("user fetched", zap.Int64("user_id", id))
	return u, nil
}
