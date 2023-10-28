package dbmigrator

import (
	"context"
	"io/fs"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"

	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
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
	db   Database
	list []DbMigration
}

func GetDbMigrationsFromFs(fsys fs.FS) ([]DbMigration, error) {
	filePaths, err := fs.Glob(fsys, "*.json")
	if err != nil {
		return nil, err
	}
	sort.Slice(filePaths, func(i, j int) bool {
		return filePaths[i] < filePaths[j]
	})
	var list []DbMigration
	upRe := regexp.MustCompile(`\.up\.json$`)
	for _, filePath := range filePaths {
		if !upRe.MatchString(filePath) {
			continue
		}
		name := upRe.ReplaceAllString(filepath.Base(filePath), "")
		downFilePath := upRe.ReplaceAllString(filePath, ".down.json")
		var upBytes []byte
		downBytes := []byte("[]")
		if slices.Contains(filePaths, downFilePath) {
			downBytes, err = fs.ReadFile(fsys, downFilePath)
			if err != nil {
				return nil, err
			}
		}
		upBytes, err = fs.ReadFile(fsys, filePath)
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
	return list, nil
}

func NewDbMigrator(db Database, list []DbMigration) *DbMigrator {
	return &DbMigrator{db: db, list: list}
}

func (d *DbMigrator) up(ctx context.Context, migration *DbMigration) (err error) {
	err = d.db.RunCommands(ctx, migration.UpScript)
	if err != nil {
		downErr := d.down(ctx, migration)
		if downErr == nil {
			return errors.Wrapf(err, "migration '%s' failed, rolled back.", migration.Name)
		} else {
			return errors.Wrapf(err, "migration '%s' failed, rolled failed to: %v", migration.Name, downErr.Error())
		}
	}
	err = d.db.InsertMigrationState(ctx, migration)
	if err != nil {
		return errors.Wrapf(err, "failed to save '%s' migration state to database", migration.Name)
	}
	return nil
}

// TODO: Consider transactions, but it requires MongoDB with replicaset
func (d *DbMigrator) down(ctx context.Context, migration *DbMigration) (err error) {
	err = d.db.RunCommands(ctx, migration.DownScript)
	if err != nil {
		return errors.Wrapf(err, "rolling back '%s' failed.", migration.Name)
	}
	err = d.db.DeleteMigrationState(ctx, migration)
	if err != nil {
		return errors.Wrapf(err, "failed to save '%s' rollback state to database", migration.Name)
	}
	return err
}

func (d *DbMigrator) GetApplied(ctx context.Context) (results []DbMigration, err error) {
	return d.db.GetAppliedMigrations(ctx)
}

func (d *DbMigrator) Plan(ctx context.Context) (plan DbPlan, err error) {
	applied, err := d.GetApplied(ctx)
	if err != nil {
		return plan, err
	}
	matchCount := 0
	for i, migration := range d.list {
		if i >= len(applied) || applied[i].Name != migration.Name || !reflect.DeepEqual(applied[i].UpScript, migration.UpScript) {
			break
		}
		matchCount++
	}

	if matchCount < len(applied) {
		plan.Downs = applied[matchCount:]
		slices.Reverse(plan.Downs)
	}
	if len(d.list) > matchCount {
		plan.Ups = d.list[matchCount:]
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
