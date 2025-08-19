package sender

import (
	"errors"
	"fmt"
	"miraclevpn/internal/repo"
	"strconv"

	"go.uber.org/zap"
)

var (
	ErrChatIDNil = errors.New("chat ID is nil")
)

type TgService struct {
	userRepo *repo.UserRepository
	sender   Sender
	logger   *zap.Logger
}

func NewTgService(userRepo *repo.UserRepository, sender Sender, logger *zap.Logger) *TgService {
	return &TgService{
		userRepo: userRepo,
		sender:   sender,
		logger:   logger,
	}
}

func (s *TgService) SendMessage(userID int64, message string) error {
	s.logger.Debug("sending telegram message", zap.Int64("user_id", userID), zap.String("message", message))

	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		s.logger.Error("failed to find user by id", zap.Int64("user_id", userID), zap.Error(err))
		return err
	}

	if user.TGChat != nil {
		chatID := strconv.Itoa(int(*(user.TGChat)))
		s.logger.Debug("sending message to TGChat", zap.Int64("user_id", userID), zap.String("chat_id", chatID))
		err := s.sender.SendMessage(chatID, message)
		if err != nil {
			s.logger.Error("failed to send message to TGChat", zap.Int64("user_id", userID), zap.String("chat_id", chatID), zap.Error(err))
			return err
		}
		s.logger.Debug("message sent to TGChat", zap.Int64("user_id", userID), zap.String("chat_id", chatID))
		return nil
	}

	s.logger.Debug("user has no TGChat", zap.Int64("user_id", userID))
	return fmt.Errorf("%w: by user %d", ErrChatIDNil, userID)
}

func (s *TgService) GetName() string {
	return s.sender.GetName()
}
