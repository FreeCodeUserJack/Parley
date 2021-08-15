package domain

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Agreement struct {
	Id 									primitive.ObjectID 	`bson:"_id" json:"agreement_id"`
	Title 							string 							`bson:"title" json:"title"`
	Description 				string 							`bson:"description" json:"description"`
	CreatedBy 					string 							`bson:"created_by" json:"created_by"`
	ArchiveId 					string 							`bson:"archive_id" json:"archive_id"`
	Participants 				[]string 						`bson:"participants" json:"participants"`
	CreateDateTime 			time.Time 					`bson:"create_datetime" json:"create_datetime"`
	LastUpdateDateTime 	time.Time 					`bson:"last_update_datetime" json:"last_update_datetime"`
	Agreement_Deadline 	Deadline 						`bson:"agreement_deadline" json:"agreement_deadline"`
	Status 							string 							`bson:"status" json:"status"`
	Public 							bool 								`bson:"public" json:"public"`
}