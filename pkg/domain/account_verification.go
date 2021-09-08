package domain

import (
	"encoding/json"
	"html"
	"strings"
	"time"
)

type AccountVerificationAlias AccountVerification
type AccountVerification struct {
	Id             string    `bson:"_id" json:"_id"`
	CreateDateTime time.Time `bson:"create_datetime" json:"-"`
	ReadDateTime   time.Time `bson:"read_datetime" json:"-"`
	UserId         string    `bson:"user_id" json:"user_id"`
	Email          string    `bson:"email" json:"email"`
	Phone          string    `bson:"phone" json:"phone"`
	Type           string    `bson:"type" json:"type"`
	OTP            string    `bson:"otp" json:"otp"`
	Status         string    `bson:"status" json:"status"`
}

func (a AccountVerification) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewJSONAccountVerification(a))
}

func (a *AccountVerification) UnmarshalJSON(data []byte) error {
	var ja JSONAccountVerification
	if err := json.Unmarshal(data, &ja); err != nil {
		return err
	}
	*a = ja.AccountVerification()
	return nil
}

func NewJSONAccountVerification(accountVerification AccountVerification) JSONAccountVerification {
	return JSONAccountVerification{
		AccountVerificationAlias(accountVerification),
		Time{accountVerification.CreateDateTime},
		Time{accountVerification.ReadDateTime},
	}
}

type JSONAccountVerification struct {
	AccountVerificationAlias
	CreateDateTime Time `json:"create_datetime"`
	ReadDateTime   Time `json:"read_datetime"`
}

func (ja JSONAccountVerification) AccountVerification() AccountVerification {
	accountVerification := AccountVerification(ja.AccountVerificationAlias)
	accountVerification.CreateDateTime = ja.CreateDateTime.Time
	accountVerification.ReadDateTime = ja.ReadDateTime.Time
	return accountVerification
}

// Sanitize
func (a *AccountVerification) Sanitize() {
	a.Id = strings.TrimSpace(html.EscapeString(a.Id))
	a.UserId = strings.TrimSpace(html.EscapeString(a.UserId))
	a.Email = strings.TrimSpace(html.EscapeString(a.Email))
	a.Phone = strings.TrimSpace(html.EscapeString(a.Phone))
	a.Type = strings.TrimSpace(html.EscapeString(a.Type))
	a.OTP = strings.TrimSpace(html.EscapeString(a.OTP))
	a.Status = strings.TrimSpace(html.EscapeString(a.Status))
}

// Validate
func (a AccountVerification) Validate() bool {
	if a.Email == "" && a.Phone == "" || a.UserId == "" {
		return false
	}

	return true
}
