package domain

import (
	"encoding/json"
	"time"
)

type NotificationAlias Notification

type Notification struct {
	Id             string    `bson:"_id" json:"_id"`
	Title          string    `bson:"title" json:"title"`
	Message        string    `bson:"message" json:"message"`
	CreateDateTime time.Time `bson:"create_datetime" json:"-"`
	ReadDateTime   time.Time `bson:"read_datetime" json:"-"`
	Status         string    `bson:"status" json:"status"`
	UserId         string    `bson:"user_id" json:"user_id"`
	ContactId      string    `bson:"contact_id" json:"contact_id"`
	AgreementId    string    `bson:"agreement_id" json:"agreement_id"`
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
