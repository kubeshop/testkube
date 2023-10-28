package dbmigrator

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"go.mongodb.org/mongo-driver/bson"
)

var (
	migration1 = DbMigration{
		Name:       "migration1",
		UpScript:   []bson.D{{{Key: "A1", Value: "VA1"}}},
		DownScript: []bson.D{{{Key: "A2", Value: "VA2"}}},
	}
	migration2 = DbMigration{
		Name:       "migration2",
		UpScript:   []bson.D{{{Key: "B1", Value: "VB1"}}},
		DownScript: []bson.D{{{Key: "B2", Value: "VB2"}}},
	}
	migration2Changed = DbMigration{
		Name:       "migration2",
		UpScript:   []bson.D{{{Key: "AC1", Value: "VAC1"}}},
		DownScript: []bson.D{{{Key: "AC2", Value: "VAC2"}}},
	}
	migration3 = DbMigration{
		Name:       "migration3",
		UpScript:   []bson.D{{{Key: "C1", Value: "VC1"}}},
		DownScript: []bson.D{{{Key: "C2", Value: "VC2"}}},
	}
)

func TestDbMigrator_GetApplied(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	db := NewMockDatabase(mockCtrl)
	migrator := NewDbMigrator(db, []DbMigration{})
	ctx := context.Background()
	expected := []DbMigration{migration1, migration2}

	db.EXPECT().GetAppliedMigrations(ctx).Return(expected, nil)

	result, err := migrator.GetApplied(ctx)

	assert.Equal(t, result, expected)
	assert.NoError(t, err)
}

func TestDbMigrator_Plan_Empty(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	db := NewMockDatabase(mockCtrl)
	migrator := NewDbMigrator(db, []DbMigration{})
	ctx := context.Background()

	db.EXPECT().GetAppliedMigrations(ctx).Return([]DbMigration{}, nil)

	result, err := migrator.Plan(ctx)

	assert.Equal(t, DbPlan{Ups: nil, Downs: nil, Total: 0}, result)
	assert.NoError(t, err)
}

func TestDbMigrator_Plan_Same(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	db := NewMockDatabase(mockCtrl)
	migrator := NewDbMigrator(db, []DbMigration{migration1})
	ctx := context.Background()

	db.EXPECT().GetAppliedMigrations(ctx).Return([]DbMigration{migration1}, nil)

	result, err := migrator.Plan(ctx)

	assert.Equal(t, DbPlan{Ups: nil, Downs: nil, Total: 0}, result)
	assert.NoError(t, err)
}

func TestDbMigrator_Plan_New(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	db := NewMockDatabase(mockCtrl)
	migrator := NewDbMigrator(db, []DbMigration{migration1, migration2})
	ctx := context.Background()

	db.EXPECT().GetAppliedMigrations(ctx).Return([]DbMigration{migration1}, nil)

	result, err := migrator.Plan(ctx)

	assert.Equal(t, DbPlan{Ups: []DbMigration{migration2}, Downs: nil, Total: 1}, result)
	assert.NoError(t, err)
}

func TestDbMigrator_Plan_Deleted(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	db := NewMockDatabase(mockCtrl)
	migrator := NewDbMigrator(db, []DbMigration{migration1})
	ctx := context.Background()

	db.EXPECT().GetAppliedMigrations(ctx).Return([]DbMigration{migration1, migration2}, nil)

	result, err := migrator.Plan(ctx)

	assert.Equal(t, DbPlan{Ups: nil, Downs: []DbMigration{migration2}, Total: 1}, result)
	assert.NoError(t, err)
}

func TestDbMigrator_Plan_Updated(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	db := NewMockDatabase(mockCtrl)
	migrator := NewDbMigrator(db, []DbMigration{migration1, migration2Changed, migration3})
	ctx := context.Background()

	db.EXPECT().GetAppliedMigrations(ctx).Return([]DbMigration{migration1, migration2, migration3}, nil)

	result, err := migrator.Plan(ctx)

	assert.Equal(t, DbPlan{Ups: []DbMigration{migration2Changed, migration3}, Downs: []DbMigration{migration3, migration2}, Total: 4}, result)
	assert.NoError(t, err)
}

func TestDbMigrator_Apply_Empty(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	db := NewMockDatabase(mockCtrl)
	migrator := NewDbMigrator(db, []DbMigration{})
	ctx := context.Background()

	db.EXPECT().GetAppliedMigrations(ctx).Return([]DbMigration{}, nil)

	err := migrator.Apply(ctx)

	assert.NoError(t, err)
}

func TestDbMigrator_Apply_Same(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	db := NewMockDatabase(mockCtrl)
	migrator := NewDbMigrator(db, []DbMigration{migration1})
	ctx := context.Background()

	db.EXPECT().GetAppliedMigrations(ctx).Return([]DbMigration{migration1}, nil)

	err := migrator.Apply(ctx)

	assert.NoError(t, err)
}

func TestDbMigrator_Apply_New(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	db := NewMockDatabase(mockCtrl)
	migrator := NewDbMigrator(db, []DbMigration{migration1, migration2})
	ctx := context.Background()

	db.EXPECT().GetAppliedMigrations(ctx).Return([]DbMigration{migration1}, nil)
	db.EXPECT().RunCommands(ctx, migration2.UpScript).Return(nil)
	db.EXPECT().InsertMigrationState(ctx, &migration2).Return(nil)

	err := migrator.Apply(ctx)

	assert.NoError(t, err)
}

func TestDbMigrator_Apply_Deleted(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	db := NewMockDatabase(mockCtrl)
	migrator := NewDbMigrator(db, []DbMigration{migration1})
	ctx := context.Background()

	db.EXPECT().GetAppliedMigrations(ctx).Return([]DbMigration{migration1, migration2}, nil)
	db.EXPECT().RunCommands(ctx, migration2.DownScript).Return(nil)
	db.EXPECT().DeleteMigrationState(ctx, &migration2).Return(nil)

	err := migrator.Apply(ctx)

	assert.NoError(t, err)
}

func TestDbMigrator_Apply_Updated(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	db := NewMockDatabase(mockCtrl)
	migrator := NewDbMigrator(db, []DbMigration{migration1, migration2Changed, migration3})
	ctx := context.Background()

	gomock.InOrder(
		db.EXPECT().GetAppliedMigrations(ctx).Return([]DbMigration{migration1, migration2, migration3}, nil),
		db.EXPECT().RunCommands(ctx, migration3.DownScript).Return(nil),
		db.EXPECT().DeleteMigrationState(ctx, &migration3).Return(nil),
		db.EXPECT().RunCommands(ctx, migration2.DownScript).Return(nil),
		db.EXPECT().DeleteMigrationState(ctx, &migration2).Return(nil),
		db.EXPECT().RunCommands(ctx, migration2Changed.UpScript).Return(nil),
		db.EXPECT().InsertMigrationState(ctx, &migration2Changed).Return(nil),
		db.EXPECT().RunCommands(ctx, migration3.UpScript).Return(nil),
		db.EXPECT().InsertMigrationState(ctx, &migration3).Return(nil),
	)

	err := migrator.Apply(ctx)

	assert.NoError(t, err)
}

func TestDbMigrator_Apply_Downgrade_On_Apply_Error(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	db := NewMockDatabase(mockCtrl)
	migrator := NewDbMigrator(db, []DbMigration{migration1, migration2})
	ctx := context.Background()

	gomock.InOrder(
		db.EXPECT().GetAppliedMigrations(ctx).Return([]DbMigration{migration1}, nil),
		db.EXPECT().RunCommands(ctx, migration2.UpScript).Return(errors.New("test-failed")),
		db.EXPECT().RunCommands(ctx, migration2.DownScript).Return(nil),
		db.EXPECT().DeleteMigrationState(ctx, &migration2).Return(nil),
	)

	err := migrator.Apply(ctx)

	assert.Error(t, err, "test-failed")
}

func TestGetDbMigrationsFromFs_Empty(t *testing.T) {
	fsys := &afero.IOFS{Fs: afero.NewMemMapFs()}
	migrations, err := GetDbMigrationsFromFs(fsys)
	assert.Equal(t, []DbMigration(nil), migrations)
	assert.NoError(t, err)
}

func TestGetDbMigrationsFromFs_Files(t *testing.T) {
	fsys := &afero.IOFS{Fs: afero.NewMemMapFs()}
	_ = afero.WriteFile(fsys.Fs, "02_file.up.json", []byte(`[{"a": "2"}]`), 0644)
	_ = afero.WriteFile(fsys.Fs, "01_file.up.json", []byte(`[{"a": "1"}]`), 0644)
	_ = afero.WriteFile(fsys.Fs, "02_file.down.json", []byte(`[{"b": "2"}]`), 0644)
	_ = afero.WriteFile(fsys.Fs, "no_suffix.json", []byte(`[{"b": "2"}]`), 0644)
	_ = afero.WriteFile(fsys.Fs, "different_ext.up.js", []byte(`[{"b": "2"}]`), 0644)
	migrations, err := GetDbMigrationsFromFs(fsys)
	assert.Equal(t, []DbMigration{
		{Name: "01_file", UpScript: []bson.D{{{Key: "a", Value: "1"}}}, DownScript: []bson.D{}},
		{Name: "02_file", UpScript: []bson.D{{{Key: "a", Value: "2"}}}, DownScript: []bson.D{{{Key: "b", Value: "2"}}}},
	}, migrations)
	assert.NoError(t, err)
}
