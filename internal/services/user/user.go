package user

import (
	"errors"
	"miraclevpn/internal/models"
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/sender"
	"strconv"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrWrongCode         = errors.New("wrong verification code")
	ErrNotFound          = errors.New("user not found")
	ErrPasswordNotEqual  = errors.New("passwords do not match")
	ErrPasswordDuplicate = errors.New("passwords are identical")
)

type UserService struct {
	userRepo *repo.UserRepository
	veriRepo *repo.VerifierRepository
	sender   *sender.TgService
	logger   *zap.Logger
}

func NewUserService(userRepo *repo.UserRepository, veriRepo *repo.VerifierRepository, sender *sender.TgService, logger *zap.Logger) *UserService {
	return &UserService{
		userRepo: userRepo,
		veriRepo: veriRepo,
		logger:   logger,
		sender:   sender,
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
	s.logger.Debug("user fetched", zap.Int64("user_id", id))
	return u, nil
}

func (s *UserService) ResetPasswordVerify(username string, code int32, newPassword string, newPasswordVerify string) error {
	u, err := s.userRepo.FindByUsername(username)
	if err != nil {
		s.logger.Error("failed to find user by username", zap.String("username", username), zap.Error(err))
		return err
	}

	userID := u.ID

	if newPassword != newPasswordVerify {
		s.logger.Warn("passwords do not match", zap.Int64("user_id", userID))
		return ErrPasswordNotEqual
	}

	if s.userRepo.CheckPassword(u.ID, newPassword) {
		return ErrPasswordDuplicate
	}

	ok, err := s.veriRepo.Verify(userID, code)
	if err != nil {
		s.logger.Error("failed to verify code", zap.Int64("user_id", userID), zap.Int32("code", code), zap.Error(err))
		return err
	}

	if !ok {
		s.logger.Warn("wrong verification code", zap.Int64("user_id", userID), zap.Int32("code", code))
		return ErrWrongCode
	}

	if err := s.userRepo.SetPassword(userID, newPassword); err != nil {
		s.logger.Error("failed to update password", zap.Int64("user_id", userID), zap.Error(err))
		return err
	}

	if err := s.veriRepo.DeleteByUserID(userID); err != nil {
		s.logger.Error("failed to delete verifier by user id", zap.Int64("user_id", userID), zap.Error(err))
		return err
	}

	s.logger.Info("password reset successfully", zap.Int64("user_id", userID))
	return nil
}

func (s *UserService) ResetPasswordSend(username string) error {
	s.logger.Debug("sending reset password verification code", zap.String("username", username))

	u, err := s.userRepo.FindByUsername(username)
	if err != nil {
		return err
	}

	userID := u.ID

	code, err := s.veriRepo.Create(userID, time.Now().Add(15*time.Minute))
	if err != nil {
		s.logger.Error("failed to create reset password verification code", zap.Int64("user_id", userID), zap.Error(err))
		return err
	}

	if err := s.sender.SendMessage(userID, "Код для изменения пароля: "+strconv.Itoa(int(code))); err != nil {
		s.logger.Error("failed to send reset password verification code", zap.Int64("user_id", userID), zap.Int32("code", code), zap.Error(err))
		return err
	}

	s.logger.Debug("reset password verification code sent", zap.Int64("user_id", userID), zap.Int32("code", code))
	return nil
}
