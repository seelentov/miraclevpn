package user

import (
	"errors"
	"miraclevpn/internal/models"
	"miraclevpn/internal/repo"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrWrongCode = errors.New("wrong verification code")
	ErrNotFound  = errors.New("user not found")
)

type UserService struct {
	userRepo *repo.UserRepository
	veriRepo *repo.VerifierRepository
	logger   *zap.Logger
}

func NewUserService(userRepo *repo.UserRepository, veriRepo *repo.VerifierRepository, logger *zap.Logger) *UserService {
	return &UserService{
		userRepo: userRepo,
		veriRepo: veriRepo,
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
	u.Password = ""
	s.logger.Info("user fetched", zap.Int64("user_id", id))
	return u, nil
}

// На будущее длля сброса пароля
// func (s *UserService) Activate(userID int64, code int32) error {
// 	s.logger.Debug("activating user", zap.Int64("user_id", userID), zap.Int32("code", code))
// 	ok, err := s.veriRepo.Verify(userID, code)
// 	if err != nil {
// 		s.logger.Error("failed to verify code", zap.Int64("user_id", userID), zap.Int32("code", code), zap.Error(err))
// 		return err
// 	}

// 	if !ok {
// 		s.logger.Warn("wrong verification code", zap.Int64("user_id", userID), zap.Int32("code", code))
// 		return ErrWrongCode
// 	}

// 	if err := s.userRepo.Activate(userID); err != nil {
// 		s.logger.Error("failed to activate user", zap.Int64("user_id", userID), zap.Error(err))
// 		return err
// 	}

// 	if err := s.veriRepo.DeleteByUserID(userID); err != nil {
// 		s.logger.Error("failed to delete verifier by user id", zap.Int64("user_id", userID), zap.Error(err))
// 		return err
// 	}

// 	s.logger.Info("user activated", zap.Int64("user_id", userID))
// 	return nil
// }
