package payment

import (
	"miraclevpn/internal/models"
	"miraclevpn/internal/repo"

	"go.uber.org/zap"
)

type AutoPaymentService struct {
	client      PaymentClient
	payRepo     *repo.PaymentRepository
	payPlanRepo *repo.PaymentPlanRepository
	userRepo    *repo.UserRepository
	logger      *zap.Logger
}

func NewAutoPaymentService(
	client PaymentClient,
	payRepo *repo.PaymentRepository,
	payPlanRepo *repo.PaymentPlanRepository,
	userRepo *repo.UserRepository,
	logger *zap.Logger,
) *AutoPaymentService {
	return &AutoPaymentService{
		client,
		payRepo,
		payPlanRepo,
		userRepo,
		logger,
	}
}

func (s *AutoPaymentService) FindForAutoPayment() ([]*models.User, error) {
	return s.userRepo.FindForAutoPayment()
}

func (s *AutoPaymentService) Process(uID, email, paymentID string, getReceipt bool) error {
	plan, err := s.payPlanRepo.FindMonthly()
	if err != nil {
		return err
	}

	yooKassaID, _, err := s.client.CreatePayment(email, plan.PayDesc, []*PaymentItem{{
		Name:     plan.PayDesc,
		Quantity: 1,
		Value:    plan.Price,
		Currency: plan.Currency,
		Vat:      plan.VatCode,
	}},
		getReceipt,
		paymentID,
		map[string]string{
			"user_id": uID,
			"email":   uID,
		},
	)

	if err != nil {
		return err
	}

	if err := s.userRepo.AddSubDays(uID, plan.Days); err != nil {
		return err
	}

	if err := s.payRepo.Create(uID, yooKassaID, plan.Days, plan.ID, true); err != nil {
		return err
	}

	return nil
}
