package domain

import (
	"time"
)

type Deadline struct {
	DeadlineDateTime 		time.Time 	`bson:"deadline_datetime" json:"deadline_datetime"`
	NotifyDateTime 			time.Time 	`bson:"notify_datetime" json:"notify_datetime"`
	CreateDateTime 			time.Time 	`bson:"create_datetime" json:"create_datetime"`
	LastUpdateDatetime 	time.Time 	`bson:"last_update_datetime" json:"last_update_datetime"`
	Status 							string 			`bson:"status" json:"status"`
}