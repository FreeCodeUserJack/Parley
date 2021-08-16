package domain

import (
	"github.com/google/uuid"
)

type Agreement struct {
	Id                 uuid.UUID `bson:"_id" json:"_id"`
	Title              string    `bson:"title" json:"title"`
	Description        string    `bson:"description" json:"description"`
	CreatedBy          string    `bson:"created_by" json:"created_by"`
	ArchiveId          string    `bson:"archive_id" json:"archive_id"`
	Participants       []string  `bson:"participants" json:"participants"`
	CreateDateTime     int64     `bson:"create_datetime" json:"create_datetime"`
	LastUpdateDateTime int64     `bson:"last_update_datetime" json:"last_update_datetime"`
	Agreement_Deadline Deadline  `bson:"agreement_deadline" json:"agreement_deadline"`
	Status             string    `bson:"status" json:"status"`
	Public             bool      `bson:"public" json:"public"`
}

// Validation
func (a Agreement) Validate() bool {
	if a.Title == "" || a.Description == "" || a.CreatedBy == "" || len(a.Participants) == 0 || a.Status == "" {
		return false
	}

	if !a.Agreement_Deadline.Validate() {
		return false
	}

	return true
}
