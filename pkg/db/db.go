package db

import (
	"context"
	"os"
	"sync"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var clientInstance *mongo.Client
var clientInstanceError error

var mongoOnce sync.Once

const (
	envConnectionString                = "MONGODB_CONN_STRING"
	envDatabaseName                    = "MONGODB_DATABASE_NAME"
	envUsersCollectionName             = "MONGODB_USERS_COLLECTION_NAME"
	envAgreementsCollectionName        = "MONGODB_AGREEMENT_COLLECTION_NAME"
	envAgreementArchiveCollectionName  = "MONGODB_AGREEMENT_ARCHIVE_COLLECTION_NAME"
	envNotificationCollectionName      = "MONGODB_NOTIFICATION_COLLECTION_NAME"
	envTokenCollectionName             = "MONGODB_TOKEN_COLLECTION_NAME"
	envReplicaSetName                  = "MONGODB_REPLICA_SET_NAME"
	envEventResponsesCollectionName    = "MONGODB_EVENT_RESPONSE_COLLECTION_NAME"
	envEmailVerificationCollectionName = "MONGODB_EMAIL_VERIFICATION_COLLECTION_NAME"
)

var (
	mongoDBConnectionString         = "mongodb://localhost:27017"
	DatabaseName                    = "parleyDB"
	UsersCollectionName             = "users"
	AgreementCollectionName         = "agreements"
	AgreementArchiveCollectionName  = "agreement_archive"
	NotificationCollectionName      = "notifications"
	TokenCollectionName             = "tokens"
	ReplicaSetName                  = "parleyset"
	EventResponseCollectionName     = "event_responses"
	EmailVerificationCollectionName = "email_verifications"
)

func init() {
	if connStr := os.Getenv(envConnectionString); connStr != "" {
		mongoDBConnectionString = connStr
	}

	if dbname := os.Getenv(envDatabaseName); dbname != "" {
		DatabaseName = dbname
	}

	if userscolname := os.Getenv(envUsersCollectionName); userscolname != "" {
		UsersCollectionName = userscolname
	}

	if agreementcolname := os.Getenv(envAgreementsCollectionName); agreementcolname != "" {
		AgreementCollectionName = agreementcolname
	}

	if agreementarchivecolname := os.Getenv(envAgreementArchiveCollectionName); agreementarchivecolname != "" {
		AgreementArchiveCollectionName = agreementarchivecolname
	}

	if notificationcolname := os.Getenv(envNotificationCollectionName); notificationcolname != "" {
		NotificationCollectionName = notificationcolname
	}

	if tokencolname := os.Getenv(envTokenCollectionName); tokencolname != "" {
		TokenCollectionName = tokencolname
	}

	if replicaname := os.Getenv(envReplicaSetName); replicaname != "" {
		ReplicaSetName = replicaname
	}

	if eventresponsescolname := os.Getenv(envEventResponsesCollectionName); eventresponsescolname != "" {
		EventResponseCollectionName = eventresponsescolname
	}

	if emailverificationcolname := os.Getenv(envEmailVerificationCollectionName); emailverificationcolname != "" {
		EmailVerificationCollectionName = emailverificationcolname
	}
}

func GetMongoClient() (*mongo.Client, error) {
	mongoOnce.Do(func() {

		// SetHosts for multi nodes in cluster
		clientOptions := options.Client().ApplyURI(mongoDBConnectionString)
		clientOptions.SetReplicaSet(ReplicaSetName)

		client, err := mongo.Connect(context.TODO(), clientOptions)
		if err != nil {
			clientInstanceError = err
		}

		err = client.Ping(context.TODO(), nil)
		if err != nil {
			clientInstanceError = err
		}

		clientInstance = client
	})

	return clientInstance, clientInstanceError
}
