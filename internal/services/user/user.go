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
	ErrBanned   = errors.New("user is banned")
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

func (s *UserService) GetUserByID(id string) (*models.User, error) {
	s.logger.Debug("getting user by id", zap.String("user_id", id))
	u, err := s.userRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.logger.Warn("user not found", zap.String("user_id", id))
			return nil, ErrNotFound
		}
		s.logger.Error("failed to get user by id", zap.String("user_id", id), zap.Error(err))
		return nil, err
	}

	if u.Banned {
		s.logger.Warn("user is banned", zap.String("user_id", id))
		return nil, ErrBanned
	}

	s.logger.Debug("user fetched", zap.String("user_id", id))
	return u, nil
}

func (s *UserService) AddDays(id string, days int) error {
	s.logger.Debug("add days", zap.String("user_id", id), zap.Int("days", days))
	if err := s.userRepo.AddSubDays(id, days); err != nil {
		return err
	}
	return nil
}

func (s *UserService) UpdatePaymentMethod(userID string, paymentID string, paymentPlanID int64) error {
	return s.userRepo.UpdatePaymentMethod(userID, paymentID, paymentPlanID)
}

func (s *UserService) RemovePaymentMethod(userID string) error {
	return s.userRepo.RemovePaymentMethod(userID)
}

func (s *UserService) UpdateEmail(uID string, email string) error {
	return s.userRepo.UpdateEmail(uID, email)
}
