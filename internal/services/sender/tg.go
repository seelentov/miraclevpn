package sender

import (
	"miraclevpn/internal/repo"
	"strconv"
)

type TgService struct {
	userRepo   *repo.UserRepository
	sender     Sender
	tgTempRepo *repo.TgTempRepository
}

func NewTgService(userRepo *repo.UserRepository, sender Sender, tgTempRepo *repo.TgTempRepository) *TgService {
	return &TgService{
		userRepo:   userRepo,
		sender:     sender,
		tgTempRepo: tgTempRepo,
	}
}

func (s *TgService) SendMessage(userID int64, message string) error {
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return err
	}

	if user.TGChat != nil {
		return s.sender.SendMessage(strconv.Itoa(int(*(user.TGChat))), message)
	}

	return s.tgTempRepo.Create(userID, message)
}
