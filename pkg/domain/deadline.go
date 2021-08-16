package domain

type Deadline struct {
	DeadlineDateTime   int64  `bson:"deadline_datetime" json:"deadline_datetime"`
	NotifyDateTime     int64  `bson:"notify_datetime" json:"notify_datetime"`
	LastUpdateDatetime int64  `bson:"last_update_datetime" json:"last_update_datetime"`
	Status             string `bson:"status" json:"status"`
}

// Validation
func (d Deadline) Validate() bool {
	if d.DeadlineDateTime == 0 || d.Status == "" {
		return false
	}

	return true
}
