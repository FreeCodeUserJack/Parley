package domain

import (
	"encoding/json"
	"time"
)

type AgreementArchiveAlias AgreementArchive

type AgreementArchive struct {
	Id             string    `bson:"_id" json:"_id"`
	AgreementData  Agreement `bson:"agreement_data" json:"agreement_data"`
	CreateDateTime time.Time `bson:"create_datetime" json:"-"`
	Info           string    `bson:"info" json:"info"`
}

func (a AgreementArchive) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewAgreementArchiveJSON(a))
}

func (a *AgreementArchive) UnmarshalJSON(data []byte) error {
	var ja AgreementArchiveJSON
	if err := json.Unmarshal(data, &ja); err != nil {
		return err
	}
	*a = ja.AgreementArchive()
	return nil
}

func NewAgreementArchiveJSON(aa AgreementArchive) AgreementArchiveJSON {
	return AgreementArchiveJSON{
		AgreementArchiveAlias(aa),
		Time{aa.CreateDateTime},
	}
}

type AgreementArchiveJSON struct {
	AgreementArchiveAlias
	CreateDateTime Time `json:"create_datetime"`
}

func (j AgreementArchiveJSON) AgreementArchive() AgreementArchive {
	agreementArchive := AgreementArchive(j.AgreementArchiveAlias)
	agreementArchive.CreateDateTime = j.CreateDateTime.Time
	return agreementArchive
}
