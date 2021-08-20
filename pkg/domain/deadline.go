package domain

import (
	"encoding/json"
	"time"
)

type DeadlineAlias Deadline

type Deadline struct {
	DeadlineDateTime   time.Time `bson:"deadline_datetime" json:"-"`
	NotifyDateTime     time.Time `bson:"notify_datetime" json:"-"`
	LastUpdateDatetime time.Time `bson:"last_update_datetime" json:"-"`
	Status             string    `bson:"status" json:"status"`
	Note               string    `bson:"note" json:"note"`
}

func (d Deadline) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewJSONDeadline(d))
}

func (d *Deadline) UnmarshalJSON(data []byte) error {
	var jd JSONDeadline
	if err := json.Unmarshal(data, &jd); err != nil {
		return err
	}
	*d = jd.Deadline()
	return nil
}

func NewJSONDeadline(deadline Deadline) JSONDeadline {
	return JSONDeadline{
		DeadlineAlias(deadline),
		Time{deadline.DeadlineDateTime},
		Time{deadline.NotifyDateTime},
		Time{deadline.LastUpdateDatetime},
	}
}

type JSONDeadline struct {
	DeadlineAlias
	DeadlineDateTime   Time `json:"deadline_datetime"`
	NotifyDateTime     Time `json:"notify_datetime"`
	LastUpdateDateTime Time `json:"last_update_datetime"`
}

func (jd JSONDeadline) Deadline() Deadline {
	deadline := Deadline(jd.DeadlineAlias)
	deadline.DeadlineDateTime = jd.DeadlineDateTime.Time
	deadline.NotifyDateTime = jd.NotifyDateTime.Time
	deadline.LastUpdateDatetime = jd.LastUpdateDateTime.Time
	return deadline
}

// Validation
func (d Deadline) Validate() bool {
	if d.DeadlineDateTime.IsZero() || d.Status == "" {
		return false
	}

	return true
}
