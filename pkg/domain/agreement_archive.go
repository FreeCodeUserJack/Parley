package domain

type AgreementArchive struct {
	Id string `bson:"_id" json:"_id"`
	AgreementData Agreement `bson:"agreement_data" json:"agreement_data"`
	CreateDateTime int64 `bson:"create_datetime" json:"create_datetime"`
	Info string `bson:"info" json:"info"`
}
