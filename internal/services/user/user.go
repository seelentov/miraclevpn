package user

import (
	"errors"
	"miraclevpn/internal/models"
	"miraclevpn/internal/repo"

	"gorm.io/gorm"
)

var (
	ErrWrongCode = errors.New("wrong verification code")
	ErrNotFound  = errors.New("user not found")
)

type UserService struct {
	userRepo *repo.UserRepository
	veriRepo *repo.VerifierRepository
}

func NewUserService(userRepo *repo.UserRepository, veriRepo *repo.VerifierRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
		veriRepo: veriRepo,
	}
}

func (s *UserService) GetUserByID(id int64) (*models.User, error) {
	u, err := s.userRepo.FindByID(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	u.Password = ""
	return u, nil
}

func (s *UserService) Activate(userID int64, code int32) error {
	ok, err := s.veriRepo.Verify(userID, code)
	if err != nil {
		return err
	}

	if !ok {
		return ErrWrongCode
	}

	if err := s.userRepo.Activate(userID); err != nil {
		return err
	}

	if err := s.veriRepo.DeleteByUserID(userID); err != nil {
		return err
	}

	return nil
}
