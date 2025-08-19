package auth

import (
	"errors"
	"fmt"
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/crypt"
	"miraclevpn/internal/services/sender"
	"strconv"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrWrongPassword     = errors.New("wrong password")
	ErrNotFound          = errors.New("user not found")
	ErrAlreadyExists     = errors.New("user already exists")
	ErrNotEqualPasswords = errors.New("passwords are not equal")
)

type AuthService struct {
	userRepo *repo.UserRepository
	veriRepo *repo.VerifierRepository

	jwtSrv    *crypt.JwtService
	senderSrv *sender.TgService

	jwtDuration time.Duration

	logger *zap.Logger
}

func NewAuthService(userRepo *repo.UserRepository, veriRepo *repo.VerifierRepository, senderSrv *sender.TgService, jwtSrv *crypt.JwtService, jwtDuration time.Duration) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		veriRepo:    veriRepo,
		senderSrv:   senderSrv,
		jwtSrv:      jwtSrv,
		jwtDuration: jwtDuration,
	}
}

func (s *AuthService) Authenticate(phone, password string) (string, error) {
	user, err := s.userRepo.FindByPhone(phone)
	if err != nil {
		s.logger.Error("failed to find user by phone", zap.String("phone", phone), zap.Error(err))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrNotFound
		} else {
			return "", err
		}
	}

	if !s.userRepo.CheckPassword(user.ID, password) {
		s.logger.Warn("wrong password", zap.String("phone", phone), zap.Int64("user_id", user.ID))
		return "", ErrWrongPassword
	}

	token, err := s.jwtSrv.GenerateToken(strconv.Itoa(int(user.ID)), s.jwtDuration)
	if err != nil {
		s.logger.Error("failed to generate token", zap.Int64("user_id", user.ID), zap.Error(err))
		return "", err
	}

	s.logger.Info("user authenticated", zap.Int64("user_id", user.ID), zap.String("phone", phone))
	return token, nil
}

func (s *AuthService) SignIn(phone, password string, checkPassword string) (token string, tgLink string, err error) {
	if password != checkPassword {
		s.logger.Warn("passwords not equal", zap.String("phone", phone))
		return "", "", ErrNotEqualPasswords
	}

	_, err = s.userRepo.FindByPhone(phone)
	if err == nil {
		s.logger.Warn("user already exists", zap.String("phone", phone))
		return "", "", ErrAlreadyExists
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Error("unexpected error on find by phone", zap.String("phone", phone), zap.Error(err))
		return "", "", err
	}

	u, err := s.userRepo.Create(phone, password)
	if err != nil {
		s.logger.Error("failed to create user", zap.String("phone", phone), zap.Error(err))
		return "", "", err
	}
	s.logger.Info("user created", zap.Int64("user_id", u.ID), zap.String("phone", phone))

	code, err := s.veriRepo.Create(u.ID, time.Now().Add(15*time.Minute))
	if err != nil {
		s.logger.Error("failed to create verifier code", zap.Int64("user_id", u.ID), zap.Error(err))
		return "", "", err
	}
	s.logger.Info("verifier code created", zap.Int64("user_id", u.ID), zap.Int32("code", code))

	token, err = s.jwtSrv.GenerateToken(strconv.Itoa(int(u.ID)), s.jwtDuration)
	if err != nil {
		s.logger.Error("failed to generate token after registration", zap.Int64("user_id", u.ID), zap.Error(err))
		return "", "", err
	}

	tgToken, err := s.jwtSrv.GenerateToken(strconv.Itoa(int(u.ID)), time.Minute*2)
	if err != nil {
		s.logger.Error("failed to generate token after registration", zap.Int64("user_id", u.ID), zap.Error(err))
		return "", "", err
	}

	s.logger.Info("user registered and token generated", zap.Int64("user_id", u.ID), zap.String("phone", phone))

	tgLink = fmt.Sprintf("https://t.me/%s?start=%s", s.senderSrv.GetName(), tgToken)
	return token, tgLink, nil
}

func (s *AuthService) Activate(userID int64, chatID int64) error {
	s.logger.Debug("activating user", zap.Int64("user_id", userID))

	if err := s.userRepo.SetTGChatID(userID, chatID); err != nil {
		s.logger.Error("failed to set chat ID", zap.Int64("user_id", userID), zap.Int64("chat_id", chatID), zap.Error(err))
		return err
	}

	if err := s.userRepo.Activate(userID); err != nil {
		s.logger.Error("failed to activate user", zap.Int64("user_id", userID), zap.Error(err))
		return err
	}

	s.logger.Info("user activated", zap.Int64("user_id", userID))
	return nil
}
