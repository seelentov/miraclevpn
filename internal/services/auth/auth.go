package auth

import (
	"errors"
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/crypt"
	"miraclevpn/internal/services/sender"
	"strconv"
	"time"

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
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrNotFound
		} else {
			return "", err
		}
	}

	if !s.userRepo.CheckPassword(user.ID, password) {
		return "", ErrWrongPassword
	}

	token, err := s.jwtSrv.GenerateToken(strconv.Itoa(int(user.ID)), s.jwtDuration)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (s *AuthService) SignIn(phone, password string, checkPassword string) (string, error) {
	if password != checkPassword {
		return "", ErrNotEqualPasswords
	}

	_, err := s.userRepo.FindByPhone(phone)
	if err == nil {
		return "", ErrAlreadyExists
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}

	u, err := s.userRepo.Create(phone, password)
	if err != nil {
		return "", err
	}

	code, err := s.veriRepo.Create(u.ID, time.Now().Add(15*time.Minute))
	if err != nil {
		return "", err
	}

	if err := s.senderSrv.SendMessage(u.ID, sender.VerifyMessage(code)); err != nil {
		return "", err
	}

	token, err := s.jwtSrv.GenerateToken(strconv.Itoa(int(u.ID)), s.jwtDuration)
	if err != nil {
		return "", err
	}

	return token, nil
}
