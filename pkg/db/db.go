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
	envConnectionString         = "MONGODB_CONN_STRING"
	envDatabaseName             = "MONGODB_DATABASE_NAME"
	envUsersCollectionName      = "MONGODB_USERS_COLLECTION_NAME"
	envAgreementsCollectionName = "MONGODB_AGREEMENT_COLLECTION_NAME"
)

var (
	mongoDBConnectionString = "mongodb://localhost:27017"
	DatabaseName            = "parleyDB"
	UsersCollectionName     = "users"
	AgreementCollectionName = "agreements"
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
}

func GetMongoClient() (*mongo.Client, error) {
	mongoOnce.Do(func() {

		// SetHosts for multi nodes in cluster
		clientOptions := options.Client().ApplyURI(mongoDBConnectionString)
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
