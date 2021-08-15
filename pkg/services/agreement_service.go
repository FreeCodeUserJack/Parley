package services

import "github.com/FreeCodeUserJack/Parley/pkg/repository"

type AgreementServiceInterface interface {
}

type agreementService struct {
	AgreementRepository repository.AgreementRepositoryInterface
}

func NewAgreementService(agreementRepo repository.AgreementRepositoryInterface) AgreementServiceInterface {
	return &agreementService{
		AgreementRepository: agreementRepo,
	}
}