package domain

type TokenDetails struct {
	UserId       string `bson:"user_id" json:"user_id"`
	AccessToken  string `bson:"access_token" json:"access_token"`
	RefreshToken string `bson:"refresh_token" json:"refresh_token"`
	AccessUuid   string `bson:"access_uuid" json:"access_uuid"`
	RefreshUuid  string `bson:"refresh_uuid" json:"refresh_uuid"`
	AtExpires    int64  `bson:"at_expires" json:"at_expires"`
	RtExpires    int64  `bson:"rt_expires" json:"rt_expires"`
}
