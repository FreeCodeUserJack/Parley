package domain

type Notification struct {
	Id             string `bson:"_id" json:"_id"`
	Title          string `bson:"title" json:"title"`
	Message        string `bson:"message" json:"message"`
	CreateDateTime int64  `bson:"create_datetime" json:"create_datetime"`
	ReadDateTime   int64  `bson:"read_datetime" json:"read_datetime"`
	Status         string `bson:"status" json:"status"`
	UserId         string `bson:"user_id" json:"user_id"`
	ContactId      string `bson:"contact_id" json:"contact_id"`
	AgreementId    string `bson:"agreement_id" json:"agreement_id"`
}
