package domain

import (
	"encoding/json"
	"html"
	"time"
)

type AgreementAlias Agreement

type Agreement struct {
	Id                         string    `bson:"_id" json:"_id"`
	Title                      string    `bson:"title" json:"title"`
	Description                string    `bson:"description" json:"description"`
	CreatedBy                  string    `bson:"created_by" json:"created_by"`
	Participants               []string  `bson:"participants" json:"participants"`
	InvitedParticipants        []string  `bson:"invited_participants" json:"invited_participants"`
	RequestedParticipants      []string  `bson:"requested_participants" json:"requested_participants"`
	PendingRemovalParticipants []string  `bson:"pending_removal_participants" json:"pending_removal_participants"`
	PendingLeaveParticipants   []string  `bson:"pending_leave_participants" json:"pending_leave_participants"`
	CreateDateTime             time.Time `bson:"create_datetime" json:"-"`
	LastUpdateDateTime         time.Time `bson:"last_update_datetime" json:"-"`
	AgreementDeadline          Deadline  `bson:"agreement_deadline" json:"agreement_deadline"`
	Status                     string    `bson:"status" json:"status"`
	Public                     string    `bson:"public" json:"public"`
	Tags                       []string  `bson:"tags" json:"tags"`
	Type                       string    `bson:"type" json:"type"`
}

func (a Agreement) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewJSONAgreement(a))
}

func (a *Agreement) UnmarshalJSON(data []byte) error {
	var ja JSONAgreement
	if err := json.Unmarshal(data, &ja); err != nil {
		return err
	}
	*a = ja.Agreement()
	return nil
}

func NewJSONAgreement(agreement Agreement) JSONAgreement {
	return JSONAgreement{
		AgreementAlias(agreement),
		Time{agreement.CreateDateTime},
		Time{agreement.LastUpdateDateTime},
	}
}

type JSONAgreement struct {
	AgreementAlias
	CreateDateTime     Time `json:"create_datetime"`
	LastUpdateDateTime Time `json:"last_update_datetime"`
}

func (ja JSONAgreement) Agreement() Agreement {
	agreement := Agreement(ja.AgreementAlias)
	agreement.CreateDateTime = ja.CreateDateTime.Time
	agreement.LastUpdateDateTime = ja.LastUpdateDateTime.Time
	return agreement
}

// Validation
func (a Agreement) Validate() bool {
	if a.Title == "" || a.Description == "" || a.CreatedBy == "" || len(a.Participants) == 0 || a.Status == "" {
		return false
	}

	if !a.AgreementDeadline.Validate() {
		return false
	}

	return true
}

// Time struct for time.Time for custom marshal/unmarshal of this field

type Time struct {
	time.Time
}

func (t Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.Time.Unix())
}

func (t *Time) UnmarshalJSON(data []byte) error {
	var i int64
	if err := json.Unmarshal(data, &i); err != nil {
		return err
	}
	t.Time = time.Unix(i, 0).UTC()
	return nil
}

// the slice array for participants of ActionAndNotification will never be passed as input and won't ever be set from json requests
func (a *Agreement) Sanitize() {
	a.Id = html.EscapeString(a.Id)
	// a.Title = html.EscapeString(a.Title)
	// a.Description = html.EscapeString(a.Description)
	a.CreatedBy = html.EscapeString(a.CreatedBy)
	a.AgreementDeadline.Sanitize()
	a.Status = html.EscapeString(a.Status)
	a.Public = html.EscapeString(a.Public)
	a.Type = html.EscapeString(a.Type)
	a.Tags = sanitizeStringSlice(a.Tags)
	a.Participants = sanitizeStringSlice(a.Participants)
	a.InvitedParticipants = sanitizeStringSlice(a.InvitedParticipants)
	a.RequestedParticipants = sanitizeStringSlice(a.RequestedParticipants)
	a.PendingRemovalParticipants = sanitizeStringSlice(a.PendingRemovalParticipants)
	a.PendingLeaveParticipants = sanitizeStringSlice(a.PendingLeaveParticipants)
}

func sanitizeStringSlice(input []string) []string {
	res := make([]string, len(input))

	for i := 0; i < len(input); i++ {
		res[i] = html.EscapeString(input[i])
	}

	return res
}
