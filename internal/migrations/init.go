package migrations

import "github.com/kubeshop/testkube/pkg/migrator"

var Migrator migrator.Migrator

func init() {
	Migrator = *migrator.NewMigrator()
}
