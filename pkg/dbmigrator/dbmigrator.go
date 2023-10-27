package dbmigrator

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"time"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/exp/slices"
)

type DbPlan struct {
	Ups   []DbMigration
	Downs []DbMigration
	Total int
}

type DbMigration struct {
	Name       string   `bson:"name"`
	UpScript   []bson.D `bson:"up"`
	DownScript []bson.D `bson:"down"`
}

type DbMigrator struct {
	db             *mongo.Database
	migrationsColl *mongo.Collection
	list           []DbMigration
}

// TODO: Consider locks
func NewDbMigrator(db *mongo.Database, collName, dirPath string) (*DbMigrator, error) {
	filePaths, err := filepath.Glob(filepath.Join(dirPath, "*.json"))
	if err != nil {
		return nil, err
	}
	var list []DbMigration
	upRe := regexp.MustCompile(`\.up\.json$`)
	for _, filePath := range filePaths {
		if !upRe.MatchString(filePath) {
			continue
		}
		name := upRe.ReplaceAllString(filepath.Base(filePath), "")
		downFilePath := upRe.ReplaceAllString(filePath, ".down.json")
		downBytes, upBytes := []byte("[]"), []byte("[]")
		if slices.Contains(filePaths, downFilePath) {
			downBytes, err = os.ReadFile(downFilePath)
			if err != nil {
				return nil, err
			}
		}
		upBytes, err = os.ReadFile(filePath)
		if err != nil {
			return nil, err
		}
		var downScript, upScript []bson.D
		err = bson.UnmarshalExtJSON(downBytes, true, &downScript)
		if err != nil {
			return nil, errors.Wrapf(err, "migration '%s' has invalid rollback commands", name)
		}
		err = bson.UnmarshalExtJSON(upBytes, true, &upScript)
		if err != nil {
			return nil, errors.Wrapf(err, "migration '%s' has invalid commands", name)
		}
		list = append(list, DbMigration{
			Name:       name,
			UpScript:   upScript,
			DownScript: downScript,
		})
	}

	return &DbMigrator{
		db:             db,
		list:           list,
		migrationsColl: db.Collection(collName),
	}, nil
}

// TODO: Consider transactions, but it requires MongoDB with replicaset
func (d *DbMigrator) up(ctx context.Context, migration *DbMigration) (err error) {
	for _, cmd := range migration.UpScript {
		err = d.db.RunCommand(ctx, cmd).Err()
		if err != nil {
			downErr := d.down(ctx, migration)
			if downErr == nil {
				return errors.Wrapf(err, "migration '%s' failed, rolled back.", migration.Name)
			} else {
				return errors.Wrapf(err, "migration '%s' failed, rolled failed to: %v", migration.Name, downErr.Error())
			}
		}
	}
	_, err = d.migrationsColl.InsertOne(ctx, bson.M{
		"name":      migration.Name,
		"up":        migration.UpScript,
		"down":      migration.DownScript,
		"timestamp": time.Now(),
	})
	if err != nil {
		return errors.Wrapf(err, "failed to save '%s' migration state to database", migration.Name)
	}
	return nil
}

// TODO: Consider transactions, but it requires MongoDB with replicaset
func (d *DbMigrator) down(ctx context.Context, migration *DbMigration) (err error) {
	for _, cmd := range migration.DownScript {
		err = d.db.RunCommand(ctx, cmd).Err()
		if err != nil {
			return errors.Wrapf(err, "rolling back '%s' failed.", migration.Name)
		}
	}
	_, err = d.migrationsColl.DeleteOne(ctx, bson.M{"name": migration.Name})
	if err != nil {
		return errors.Wrapf(err, "failed to save '%s' rollback state to database", migration.Name)
	}
	return err
}

func (d *DbMigrator) GetApplied(ctx context.Context) (results []DbMigration, err error) {
	cursor, err := d.migrationsColl.Find(ctx, bson.M{}, &options.FindOptions{Sort: bson.M{"name": 1}})
	if err != nil {
		return nil, err
	}
	err = cursor.All(ctx, &results)
	return results, err
}

func (d *DbMigrator) Plan(ctx context.Context) (plan DbPlan, err error) {
	applied, err := d.GetApplied(ctx)
	if err != nil {
		return plan, err
	}
	plan.Ups = d.list[len(applied):]
	for i, migration := range applied {
		if i > len(d.list) {
			plan.Ups = d.list[i:]
			break
		}
		if d.list[i].Name != migration.Name || !reflect.DeepEqual(d.list[i].UpScript, migration.UpScript) {
			plan.Ups = d.list[i:]
			plan.Downs = applied[i:]
			break
		}
	}
	plan.Total = len(plan.Ups) + len(plan.Downs)
	return plan, err
}

func (d *DbMigrator) Apply(ctx context.Context) error {
	plan, err := d.Plan(ctx)
	if err != nil {
		return err
	}
	for _, migration := range plan.Downs {
		err = d.down(ctx, &migration)
		if err != nil {
			return err
		}
	}
	for _, migration := range plan.Ups {
		err = d.up(ctx, &migration)
		if err != nil {
			return err
		}
	}
	return nil
}
