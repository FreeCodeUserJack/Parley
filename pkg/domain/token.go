package domain

import "time"

type Token struct {
	Id          string    `bson:"_id" json:"_id"`
	ExpiresAt   time.Time `bson:"expires_at" json:"expires_at"`
	TokenString string    `bson:"token_string" json:"token_string"`
	UserId      string    `bson:"user_id" json:"user_id"`
}
