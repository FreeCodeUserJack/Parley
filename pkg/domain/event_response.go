package domain

import (
	"encoding/json"
	"html"
	"strings"
	"time"
)

type EventResponseAlias EventResponse
type EventResponse struct {
	Id             string    `bson:"_id" json:"_id"`
	AgreementId    string    `bson:"agreement_id" json:"agreement_id"`
	Status         string    `bson:"status" json:"status"`
	Message        string    `bson:"message" json:"message"`
	Response       string    `bson:"response" json:"response"`
	CreateDateTime time.Time `bson:"create_datetime" json:"-"`
	UserId         string    `bson:"user_id" json:"user_id"`
	UserFirstName  string    `bson:"user_first_name" json:"user_first_name"`
}

func (e EventResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewJSONEventResponse(e))
}

func (e *EventResponse) UnmarshalJSON(data []byte) error {
	var ja JSONEventResponse
	if err := json.Unmarshal(data, &ja); err != nil {
		return err
	}
	*e = ja.EventResponse()
	return nil
}

func NewJSONEventResponse(eventResponse EventResponse) JSONEventResponse {
	return JSONEventResponse{
		EventResponseAlias(eventResponse),
		Time{eventResponse.CreateDateTime},
	}
}

type JSONEventResponse struct {
	EventResponseAlias
	CreateDateTime Time `json:"create_datetime"`
}

func (ja JSONEventResponse) EventResponse() EventResponse {
	eventResponse := EventResponse(ja.EventResponseAlias)
	eventResponse.CreateDateTime = ja.CreateDateTime.Time
	return eventResponse
}

// Validate
func (e EventResponse) Validate() bool {
	if e.AgreementId == "" || e.Response == "" {
		return false
	}
	return true
}

// Sanitize
func (e *EventResponse) Sanitize() {
	e.Id = strings.TrimSpace(html.EscapeString(e.Id))
	e.AgreementId = strings.TrimSpace(html.EscapeString(e.AgreementId))
	e.Message = strings.TrimSpace(html.EscapeString(e.Message))
	e.Response = strings.TrimSpace(html.EscapeString(e.Response))
	e.Status = strings.TrimSpace(html.EscapeString(e.Status))
	e.UserFirstName = strings.TrimSpace(html.EscapeString(e.UserFirstName))
	e.UserId = strings.TrimSpace(html.EscapeString(e.UserId))
}
