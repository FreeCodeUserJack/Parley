package domain

import (
	"encoding/json"
	"html"
	"strings"
	"time"
)

type NotificationAlias Notification

type Notification struct {
	Id               string    `bson:"_id" json:"_id"`
	Title            string    `bson:"title" json:"title"`
	Message          string    `bson:"message" json:"message"`
	CreateDateTime   time.Time `bson:"create_datetime" json:"-"`
	ReadDateTime     time.Time `bson:"read_datetime" json:"-"`       // set when either dismissed or responded
	Status           string    `bson:"status" json:"status"`         // new or old
	UserId           string    `bson:"user_id" json:"user_id"`       // who the notification is for
	ContactId        string    `bson:"contact_id" json:"contact_id"` // who is sending the notification
	ContactFirstName string    `bson:"contact_first_name" json:"contact_first_name"`
	AgreementId      string    `bson:"agreement_id" json:"agreement_id"`
	AgreementTitle   string    `bson:"agreement_title" json:"agreement_title"`
	Response         string    `bson:"response" json:"response"` // accept or decline
	Type             string    `bson:"type" json:"type"`         // notify or requires_response
	Action           string    `bson:"action" json:"action"`     // specific action e.g. invite
}

func (n Notification) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewJSONNotification(n))
}

func (n *Notification) UnmarshalJSON(data []byte) error {
	var jn JSONNotification
	if err := json.Unmarshal(data, &jn); err != nil {
		return err
	}
	*n = jn.Notification()
	return nil
}

func NewJSONNotification(notification Notification) JSONNotification {
	return JSONNotification{
		NotificationAlias(notification),
		Time{notification.CreateDateTime},
		Time{notification.ReadDateTime},
	}
}

type JSONNotification struct {
	NotificationAlias
	CreateDateTime Time `json:"create_datetime"`
	ReadDateTime   Time `json:"read_datetime"`
}

func (jn JSONNotification) Notification() Notification {
	notification := Notification(jn.NotificationAlias)
	notification.CreateDateTime = jn.CreateDateTime.Time
	notification.ReadDateTime = jn.ReadDateTime.Time
	return notification
}

func (n *Notification) Sanitize() {
	n.Id = html.EscapeString(n.Id)
	n.Action = strings.TrimSpace(html.EscapeString(n.Action))
	n.AgreementId = strings.TrimSpace(html.EscapeString(n.AgreementId))
	n.AgreementTitle = strings.TrimSpace(html.EscapeString(n.AgreementTitle))
	n.ContactFirstName = strings.TrimSpace(html.EscapeString(n.ContactFirstName))
	n.ContactId = strings.TrimSpace(html.EscapeString(n.ContactId))
	// n.Message = html.EscapeString(n.Message)
	n.Response = strings.TrimSpace(html.EscapeString(n.Response))
	n.Status = strings.TrimSpace(html.EscapeString(n.Status))
	n.Title = strings.TrimSpace(html.EscapeString(n.Title))
	n.Type = strings.TrimSpace(html.EscapeString(n.Type))
	n.UserId = strings.TrimSpace(html.EscapeString(n.UserId))
}
