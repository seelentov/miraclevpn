// Package auth provides authentication services for the application.
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

func NewAuthService(userRepo *repo.UserRepository, veriRepo *repo.VerifierRepository, senderSrv *sender.TgService, jwtSrv *crypt.JwtService, jwtDuration time.Duration, logger *zap.Logger) *AuthService {
	return &AuthService{
		userRepo:    userRepo,
		veriRepo:    veriRepo,
		senderSrv:   senderSrv,
		jwtSrv:      jwtSrv,
		logger:      logger,
		jwtDuration: jwtDuration,
	}
}

func (s *AuthService) Authenticate(username, password string) (token string, tgLink string, err error) {
	user, err := s.userRepo.FindByUsername(username)
	if err != nil {
		s.logger.Error("failed to find user by username", zap.String("username", username), zap.Error(err))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", "", ErrNotFound
		} else {
			return "", "", err
		}
	}

	if !s.userRepo.CheckPassword(user.ID, password) {
		s.logger.Warn("wrong password", zap.String("username", username), zap.Int64("user_id", user.ID))
		return "", "", ErrWrongPassword
	}

	token, err = s.GenerateToken(user.ID)
	if err != nil {
		s.logger.Error("failed to generate token", zap.Int64("user_id", user.ID), zap.Error(err))
		return "", "", err
	}

	if !user.Active {
		s.logger.Debug("user not activated", zap.Int64("user_id", user.ID), zap.String("username", username))

		tgToken, err := s.jwtSrv.GenerateToken(strconv.Itoa(int(user.ID)), time.Minute*2)
		if err != nil {
			s.logger.Error("failed to generate token after registration", zap.Int64("user_id", user.ID), zap.Error(err))
			return "", "", err
		}

		tgLink = fmt.Sprintf("https://t.me/%s?text=%s", s.senderSrv.GetName(), tgToken)
	}

	s.logger.Debug("user authenticated", zap.Int64("user_id", user.ID), zap.String("username", username))
	return token, tgLink, nil
}

func (s *AuthService) SignUp(username, password string, checkPassword string) (token string, tgLink string, err error) {
	if password != checkPassword {
		s.logger.Warn("passwords not equal", zap.String("username", username))
		return "", "", ErrNotEqualPasswords
	}

	_, err = s.userRepo.FindByUsername(username)
	if err == nil {
		s.logger.Warn("user already exists", zap.String("username", username))
		return "", "", ErrAlreadyExists
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Error("unexpected error on find by username", zap.String("username", username), zap.Error(err))
		return "", "", err
	}

	u, err := s.userRepo.Create(username, password)
	if err != nil {
		s.logger.Error("failed to create user", zap.String("username", username), zap.Error(err))
		return "", "", err
	}
	s.logger.Debug("user created", zap.Int64("user_id", u.ID), zap.String("username", username))

	code, err := s.veriRepo.Create(u.ID, time.Now().Add(15*time.Minute))
	if err != nil {
		s.logger.Error("failed to create verifier code", zap.Int64("user_id", u.ID), zap.Error(err))
		return "", "", err
	}
	s.logger.Debug("verifier code created", zap.Int64("user_id", u.ID), zap.Int32("code", code))

	token, err = s.GenerateToken(u.ID)
	if err != nil {
		s.logger.Error("failed to generate token after registration", zap.Int64("user_id", u.ID), zap.Error(err))
		return "", "", err
	}

	tgToken, err := s.jwtSrv.GenerateToken(strconv.Itoa(int(u.ID)), time.Minute*2)
	if err != nil {
		s.logger.Error("failed to generate token after registration", zap.Int64("user_id", u.ID), zap.Error(err))
		return "", "", err
	}

	s.logger.Debug("user registered and token generated", zap.Int64("user_id", u.ID), zap.String("username", username))

	tgLink = fmt.Sprintf("https://t.me/%s?text=%s", s.senderSrv.GetName(), tgToken)
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

	s.logger.Debug("user activated", zap.Int64("user_id", userID))
	return nil
}

func (s *AuthService) GenerateToken(userID int64) (string, error) {
	token, err := s.jwtSrv.GenerateToken(strconv.Itoa(int(userID)), s.jwtDuration)
	if err != nil {
		s.logger.Error("failed to generate token after registration", zap.Int64("user_id", userID), zap.Error(err))
		return "", err
	}
	return token, nil
}
