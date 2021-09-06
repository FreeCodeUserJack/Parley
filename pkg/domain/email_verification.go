package domain

import (
	"encoding/json"
	"html"
	"strings"
	"time"
)

type EmailVerificationAlias EmailVerification
type EmailVerification struct {
	Id             string    `bson:"_id" json:"_id"`
	CreateDateTime time.Time `bson:"create_datetime" json:"-"`
	ReadDateTime   time.Time `bson:"read_datetime" json:"-"`
	UserId         string    `bson:"user_id" json:"user_id"`
	Email          string    `bson:"email" json:"email"`
}

func (e EmailVerification) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewJSONEmailVerification(e))
}

func (e *EmailVerification) UnmarshalJSON(data []byte) error {
	var je JSONEmailVerification
	if err := json.Unmarshal(data, &je); err != nil {
		return err
	}
	*e = je.EmailVerification()
	return nil
}

func NewJSONEmailVerification(emailVerification EmailVerification) JSONEmailVerification {
	return JSONEmailVerification{
		EmailVerificationAlias(emailVerification),
		Time{emailVerification.CreateDateTime},
		Time{emailVerification.ReadDateTime},
	}
}

type JSONEmailVerification struct {
	EmailVerificationAlias
	CreateDateTime Time `json:"create_datetime"`
	ReadDateTime   Time `json:"read_datetime"`
}

func (je JSONEmailVerification) EmailVerification() EmailVerification {
	emailVerification := EmailVerification(je.EmailVerificationAlias)
	emailVerification.CreateDateTime = je.CreateDateTime.Time
	emailVerification.ReadDateTime = je.ReadDateTime.Time
	return emailVerification
}

// Sanitize
func (e *EmailVerification) Sanitize() {
	e.Id = strings.TrimSpace(html.EscapeString(e.Id))
	e.UserId = strings.TrimSpace(html.EscapeString(e.UserId))
	e.Email = strings.TrimSpace(html.EscapeString(e.Email))
}

// Validate
func (e EmailVerification) Validate() bool {
	if e.Email == "" || e.UserId == "" {
		return false
	}

	return true
}
