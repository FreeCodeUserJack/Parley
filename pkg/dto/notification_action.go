package dto

import "github.com/FreeCodeUserJack/Parley/pkg/domain"

type NotificationAction struct {
	Action       string              `json:"action"`
	Notification domain.Notification `json:"notification"`
}
