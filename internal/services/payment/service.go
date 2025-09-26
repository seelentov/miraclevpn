package payment

import (
	"errors"
	"fmt"
	"miraclevpn/internal/models"
	"miraclevpn/internal/repo"
	"miraclevpn/internal/services/crypt"
	"strconv"
	"time"

	"go.uber.org/zap"
)

var (
	ErrInvalidToken = errors.New("invalid token")
)

type PaymentService struct {
	client      PaymentClient
	payRepo     *repo.PaymentRepository
	payPlanRepo *repo.PaymentPlanRepository
	jwtService  *crypt.JwtService
	logger      *zap.Logger
}

func NewPaymentService(client PaymentClient, payRepo *repo.PaymentRepository, payPlanRepo *repo.PaymentPlanRepository, jwtService *crypt.JwtService, logger *zap.Logger) *PaymentService {
	return &PaymentService{
		client:      client,
		payRepo:     payRepo,
		payPlanRepo: payPlanRepo,
		jwtService:  jwtService,
		logger:      logger,
	}
}

func (s *PaymentService) Create(uID, email string, plan *models.PaymentPlan, getReceipt bool) (payURL string, err error) {
	token, err := s.makeToken(uID, plan.ID)
	if err != nil {
		return "", err
	}

	yooKassaID, payURL, err := s.client.CreatePayment(email, plan.PayDesc, []*PaymentItem{{
		Name:     plan.PayDesc,
		Quantity: 1,
		Value:    plan.Price,
		Currency: plan.Currency,
		Vat:      plan.VatCode,
	}},
		getReceipt,
		"",
		map[string]string{
			"token":   token,
			"user_id": uID,
			"email":   uID,
		},
	)
	if err != nil {
		s.logger.Error("failed to create payment", zap.String("user_id", uID), zap.Error(err))
		return "", err
	}

	s.logger.Info("payment created", zap.String("user_id", uID), zap.String("yoo_kassa_id", yooKassaID), zap.String("pay_url", payURL))
	return payURL, s.payRepo.Create(uID, yooKassaID, plan.Days, plan.ID, false)
}

func (s *PaymentService) Find(yooKassaID string) (*models.Payment, error) {
	return s.payRepo.FindByYooKassaID(yooKassaID)
}

func (s *PaymentService) Done(yooKassaID string) error {
	return s.payRepo.Done(yooKassaID)
}

func (s *PaymentService) FindPlanByID(planID int64) (*models.PaymentPlan, error) {
	return s.payPlanRepo.FindByID(planID)
}

func (s *PaymentService) ValidateToken(token string, userID string, planID int64) error {
	claims, err := s.jwtService.ParseToken(token)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}

	data := claims.Data

	if data["user_id"] != userID || data["plan"] != strconv.Itoa(int(planID)) {
		return ErrInvalidToken
	}

	return nil
}

func (s *PaymentService) makeToken(userID string, planID int64) (string, error) {
	token, err := s.jwtService.GenerateToken(map[string]string{
		"user_id": userID,
		"plan":    strconv.Itoa(int(planID)),
	}, time.Minute*10)
	return token, err
}
