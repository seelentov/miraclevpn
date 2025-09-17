package payment

import (
	"miraclevpn/internal/models"
	"miraclevpn/internal/repo"

	"go.uber.org/zap"
)

type PaymentService struct {
	client      PaymentClient
	payRepo     *repo.PaymentRepository
	payPlanRepo *repo.PaymentPlanRepository
	logger      *zap.Logger
}

func NewPaymentService(client PaymentClient, payRepo *repo.PaymentRepository, payPlanRepo *repo.PaymentPlanRepository, logger *zap.Logger) *PaymentService {
	return &PaymentService{
		client:      client,
		payRepo:     payRepo,
		payPlanRepo: payPlanRepo,
		logger:      logger,
	}
}

func (s *PaymentService) Create(uID, email string, plan *models.PaymentPlan, getReceipt bool) (payURL string, err error) {
	yooKassaID, payURL, err := s.client.CreatePayment(email, plan.PayDesc, []*PaymentItem{{
		Name:     plan.Desc,
		Quantity: 1,
		Value:    plan.Price,
		Currency: plan.Currency,
		Vat:      plan.VatCode,
	}}, getReceipt)
	if err != nil {
		s.logger.Error("failed to create payment", zap.String("user_id", uID), zap.Error(err))
		return "", err
	}

	s.logger.Info("payment created", zap.String("user_id", uID), zap.String("yoo_kassa_id", yooKassaID), zap.String("pay_url", payURL))
	return payURL, s.payRepo.Create(uID, yooKassaID, plan.Days)
}

func (s *PaymentService) Find(yooKassaID string) (*models.Payment, error) {
	return s.payRepo.FindByYooKassaID(yooKassaID)
}

func (s *PaymentService) Delete(yooKassaID string) error {
	return s.payRepo.Delete(yooKassaID)
}

func (s *PaymentService) FindPlanByID(planID int64) (*models.PaymentPlan, error) {
	return s.payPlanRepo.FindByID(planID)
}
