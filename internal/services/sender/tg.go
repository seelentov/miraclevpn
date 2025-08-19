package sender

import (
	"miraclevpn/internal/repo"
	"strconv"

	"go.uber.org/zap"
)

type TgService struct {
	userRepo   *repo.UserRepository
	sender     Sender
	tgTempRepo *repo.TgTempRepository
	logger     *zap.Logger
}

func NewTgService(userRepo *repo.UserRepository, sender Sender, tgTempRepo *repo.TgTempRepository, logger *zap.Logger) *TgService {
	return &TgService{
		userRepo:   userRepo,
		sender:     sender,
		tgTempRepo: tgTempRepo,
		logger:     logger,
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
		s.logger.Info("sending message to TGChat", zap.Int64("user_id", userID), zap.String("chat_id", chatID))
		err := s.sender.SendMessage(chatID, message)
		if err != nil {
			s.logger.Error("failed to send message to TGChat", zap.Int64("user_id", userID), zap.String("chat_id", chatID), zap.Error(err))
			return err
		}
		s.logger.Info("message sent to TGChat", zap.Int64("user_id", userID), zap.String("chat_id", chatID))
		return nil
	}

	s.logger.Info("user has no TGChat, saving temp message", zap.Int64("user_id", userID))
	err = s.tgTempRepo.Create(userID, message)
	if err != nil {
		s.logger.Error("failed to save temp message", zap.Int64("user_id", userID), zap.Error(err))
		return err
	}
	s.logger.Info("temp message saved", zap.Int64("user_id", userID))
	return nil
}
