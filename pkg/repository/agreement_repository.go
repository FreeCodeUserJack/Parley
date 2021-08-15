package repository

type AgreementRepositoryInterface interface {
}

type agreementRepository struct {
}

func NewAgreementRepository() AgreementRepositoryInterface {
	return &agreementRepository{}
}