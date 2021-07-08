package storage

import (
	"github.com/kelseyhightower/envconfig"
)

type MongoConfig struct {
	DSN string `envoconfig:"MONGO_DSN"`
}

var Config MongoConfig

func init() {
	envconfig.Process("mongo", &Config)

}

func GetMongoDataBase() {

}
