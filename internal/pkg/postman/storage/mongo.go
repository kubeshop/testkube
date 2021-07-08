package storage

import (
	"context"
	"time"

	"github.com/kelseyhightower/envconfig"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const DatabaseName = "postman-executor"

type MongoConfig struct {
	DSN string `envconfig:"MONGO_DSN" default:"mongodb://localhost:27017"`
}

var Config MongoConfig

func init() {
	envconfig.Process("mongo", &Config)
}

func GetMongoDataBase() (db *mongo.Database, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(Config.DSN))
	if err != nil {
		return nil, err
	}

	return client.Database(DatabaseName), nil
}
