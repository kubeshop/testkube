package dbmigrator

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

//go:generate mockgen -destination=./mock_database.go -package=dbmigrator "github.com/kubeshop/testkube/pkg/dbmigrator" Database
type Database interface {
	RunCommands(ctx context.Context, commands []bson.D) error
	InsertMigrationState(ctx context.Context, migration *DbMigration) error
	DeleteMigrationState(ctx context.Context, migration *DbMigration) error
	GetAppliedMigrations(ctx context.Context) ([]DbMigration, error)
}

type database struct {
	db             *mongo.Database
	migrationsColl *mongo.Collection
}

// TODO: Consider locks
func NewDatabase(db *mongo.Database, migrationsColl string) Database {
	return &database{db: db, migrationsColl: db.Collection(migrationsColl)}
}

// TODO: Consider transactions, but it requires MongoDB with replicaset
func (d *database) RunCommands(ctx context.Context, commands []bson.D) error {
	for _, cmd := range commands {
		err := d.db.RunCommand(ctx, cmd).Err()
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *database) InsertMigrationState(ctx context.Context, migration *DbMigration) error {
	_, err := d.migrationsColl.InsertOne(ctx, bson.M{
		"name":      migration.Name,
		"up":        migration.UpScript,
		"down":      migration.DownScript,
		"timestamp": time.Now(),
	})
	return err
}

func (d *database) DeleteMigrationState(ctx context.Context, migration *DbMigration) error {
	_, err := d.migrationsColl.DeleteOne(ctx, bson.M{"name": migration.Name})
	return err
}

func (d *database) GetAppliedMigrations(ctx context.Context) (results []DbMigration, err error) {
	cursor, err := d.migrationsColl.Find(ctx, bson.M{}, &options.FindOptions{Sort: bson.M{"name": 1}})
	if err != nil {
		return nil, err
	}
	err = cursor.All(ctx, &results)
	return results, err
}
