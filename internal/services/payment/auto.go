package payment

import (
	"miraclevpn/internal/models"
	"miraclevpn/internal/repo"
	"time"

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

type UserForPayment struct {
	User   *models.User
	Plan   *models.PaymentPlan
	Payday time.Time
}

func (s *AutoPaymentService) FindForPayment() ([]*UserForPayment, error) {
	users, err := s.userRepo.FindForPayment()
	if err != nil {
		return nil, err
	}

	res := make([]*UserForPayment, 0)

	for _, user := range users {
		if user.PaymentPlanID == nil {
			s.logger.Error("cant calculate autopayment: paymentPlanID is nil", zap.String("user_id", user.ID))
			continue
		}

		if user.PaymentID == nil {
			s.logger.Error("cant calculate autopayment: PaymentID is nil", zap.String("user_id", user.ID))
			continue
		}

		if user.LastPaymentAt == nil {
			s.logger.Error("cant calculate autopayment: LastPaymentAt is nil", zap.String("user_id", user.ID))
			continue
		}

		plan, err := s.payPlanRepo.FindByID(*(user.PaymentPlanID))
		if err != nil {
			s.logger.Error("cant calculate autopayment", zap.Error(err), zap.String("user_id", user.ID))
			continue
		}

		payday := user.LastPaymentAt.Add(time.Hour * 24 * time.Duration(plan.Days)).Add(time.Hour * -1)

		if payday.Before(time.Now()) {
			res = append(res, &UserForPayment{
				User:   user,
				Payday: payday,
				Plan:   plan,
			})
		}
	}

	return res, nil
}

func (s *AutoPaymentService) Process(uID, email, paymentID string, plan *models.PaymentPlan, getReceipt bool) error {
	yooKassaID, _, err := s.client.CreatePayment(email, plan.PayDesc, []*PaymentItem{{
		Name:     plan.PayDesc,
		Quantity: 1,
		Value:    plan.Price,
		Currency: plan.Currency,
		Vat:      plan.VatCode,
	}},
		"",
		getReceipt,
		paymentID,
	)

	if err != nil {
		return err
	}

	if err := s.payRepo.Create(uID, yooKassaID, plan.Days, plan.ID); err != nil {
		return err
	}

	if err := s.userRepo.AddSubDays(uID, plan.Days); err != nil {
		return err
	}

	if err := s.payRepo.Done(yooKassaID); err != nil {
		return err
	}

	return nil
}
