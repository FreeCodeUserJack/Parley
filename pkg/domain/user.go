package domain

import (
	"encoding/json"
	"html"
	"strings"
	"time"
)

type UserAlias User

type User struct {
	Id                       string    `bson:"_id" json:"_id"`
	FirstName                string    `bson:"first_name" json:"first_name"`
	LastName                 string    `bson:"last_name" json:"last_name"`
	DOB                      time.Time `bson:"dob" json:"-"`
	Phone                    string    `bson:"phone" json:"phone"`
	Email                    string    `bson:"email" json:"email"`
	Password                 string    `bson:"password" json:"password"`
	Agreements               []string  `bson:"agreements" json:"agreements"`
	InvitedAgreements        []string  `bson:"invited_agreements" json:"invited_agreements"`
	RequestedAgreements      []string  `bson:"requested_agreements" json:"requested_agreements"`
	PendingAgreementRemovals []string  `bson:"pending_agreement_removals" json:"pending_agreement_removals"`
	PendingLeaveAgreements   []string  `bson:"pending_leave_agreements" json:"pending_leave_agreements"`
	CreateDateTime           time.Time `bson:"create_datetime" json:"-"`
	LastUpdateDateTime       time.Time `bson:"last_update_datetime" json:"-"`
	Role                     string    `bson:"role" json:"role"`
	Status                   string    `bson:"status" json:"status"`
	Friends                  []string  `bson:"friends" json:"friends"`
	Public                   string    `bson:"public" json:"public"`
	PendingFriendRequests    []string  `bson:"pending_friend_requests" json:"pending_friend_requests"`
	SentFriendRequests       []string  `bson:"sent_friend_requests" json:"sent_friend_requests"`
}

func (u User) MarshalJSON() ([]byte, error) {
	u.Password = ""
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

// Validate : Validation
func (u User) Validate() bool {
	if u.DOB.After(time.Now().UTC()) {
		return false
	}

	if u.Email == "" || u.Public == "" || u.Role == "" || u.Status == "" {
		return false
	}

	return true
}

func (u *User) Sanitize() {
	u.Id = html.EscapeString(u.Id)
	u.FirstName = strings.TrimSpace(html.EscapeString(u.FirstName))
	u.LastName = strings.TrimSpace(html.EscapeString(u.LastName))
	u.Phone = strings.TrimSpace(html.EscapeString(u.Phone))
	u.Role = strings.TrimSpace(html.EscapeString(u.Role))
	u.Status = strings.TrimSpace(html.EscapeString(u.Status))
	u.Public = strings.TrimSpace(html.EscapeString(u.Public))
	u.Friends = SanitizeStringSlice(u.Friends)
	u.PendingFriendRequests = SanitizeStringSlice(u.PendingFriendRequests)
	u.Agreements = SanitizeStringSlice(u.Agreements)
	u.InvitedAgreements = SanitizeStringSlice(u.InvitedAgreements)
	u.RequestedAgreements = SanitizeStringSlice(u.RequestedAgreements)
	u.PendingAgreementRemovals = SanitizeStringSlice(u.PendingAgreementRemovals)
	u.PendingLeaveAgreements = SanitizeStringSlice(u.PendingLeaveAgreements)
	u.SentFriendRequests = SanitizeStringSlice(u.SentFriendRequests)
}
