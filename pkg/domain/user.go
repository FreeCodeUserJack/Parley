package domain

import (
	"encoding/json"
	"html"
	"time"
)

type UserAlias User

type User struct {
	Id                       string    `bson:"_id" json:"_id"`
	FirstName                string    `bson:"first_name" json:"first_name"`
	LastName                 string    `bson:"last_name" json:"last_name"`
	DOB                      time.Time `bson:"dob" json:"-"`
	Email                    string    `bson:"email" json:"email"`
	Password                 string    `bson:"password" json:"-"`
	Agreements               []string  `bson:"agreements" json:"agreements"`
	InvitedAgreements        []string  `bson:"invited_agreements" json:"invited_agreements"`
	RequestedAgreements      []string  `bson:"requested_agreements" json:"requested_agreements"`
	PendingAgreementRemovals []string  `bson:"pending_agreement_removals" json:"pending_agreement_removals"`
	PendingLeaveAgreements   []string  `bson:"pending_leave_agreements" json:"pending_leave_agreements"`
	Notifications            []string  `bson:"notifications" json:"notifications"`
	CreateDateTime           time.Time `bson:"create_datetime" json:"-"`
	LastUpdateDateTime       time.Time `bson:"last_update_datetime" json:"-"`
	Role                     string    `bson:"role" json:"role"`
	Status                   string    `bson:"status" json:"status"`
	Friends                  []string  `bson:"friends" json:"friends"`
	Public                   string    `bson:"public" json:"public"`
	PendingFriendRequests    []string  `bson:"pending_friend_requests" json:"pending_friend_requests"`
}

func (u User) MarshalJSON() ([]byte, error) {
	return json.Marshal(NewJSONUser(u))
}

func (u *User) UnmarshalJSON(data []byte) error {
	var ju JSONUser
	if err := json.Unmarshal(data, &ju); err != nil {
		return err
	}
	*u = ju.User()
	return nil
}

func NewJSONUser(user User) JSONUser {
	return JSONUser{
		UserAlias(user),
		Time{user.DOB},
		Time{user.CreateDateTime},
		Time{user.LastUpdateDateTime},
	}
}

type JSONUser struct {
	UserAlias
	DOB                Time `json:"dob"`
	CreateDateTime     Time `json:"create_datetime"`
	LastUpdateDateTime Time `json:"last_update_datetime"`
}

func (ju JSONUser) User() User {
	user := User(ju.UserAlias)
	user.DOB = ju.DOB.Time
	user.CreateDateTime = ju.CreateDateTime.Time
	user.LastUpdateDateTime = ju.LastUpdateDateTime.Time
	return user
}

// Validation
func (u User) Validate() bool {
	if u.DOB.After(time.Now().UTC()) {
		return false
	}

	if u.Email == "" || u.Public == "" || u.Role == "" || u.Status == "" {
		return false
	}

	return true
}

// Sanitize
func (u *User) Sanitize() {
	u.Id = html.EscapeString(u.Id)
	u.FirstName = html.EscapeString(u.FirstName)
	u.LastName = html.EscapeString(u.LastName)
	u.Role = html.EscapeString(u.Role)
	u.Status = html.EscapeString(u.Status)
	u.Public = html.EscapeString(u.Public)
	u.Friends = sanitizeStringSlice(u.Friends)
	u.PendingFriendRequests = sanitizeStringSlice(u.PendingFriendRequests)
	u.Agreements = sanitizeStringSlice(u.Agreements)
	u.InvitedAgreements = sanitizeStringSlice(u.InvitedAgreements)
	u.RequestedAgreements = sanitizeStringSlice(u.RequestedAgreements)
	u.PendingAgreementRemovals = sanitizeStringSlice(u.PendingAgreementRemovals)
	u.PendingLeaveAgreements = sanitizeStringSlice(u.PendingLeaveAgreements)
}
