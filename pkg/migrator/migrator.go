package migrator

import (
	"fmt"
	"strings"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/process"
	"github.com/kubeshop/testkube/pkg/version"
	"go.uber.org/zap"
)

type Migration interface {
	Migrate() error
	Version() string
	Info() string
}

func NewMigrator() *Migrator {
	return &Migrator{
		Log: log.DefaultLogger,
	}
}

type Migrator struct {
	Migrations []Migration
	Log        *zap.SugaredLogger
}

func (m *Migrator) Add(migration Migration) {
	m.Migrations = append(m.Migrations, migration)
}

func (m *Migrator) GetValidMigrations(currentVersion string) (migrations []Migration) {
	for _, migration := range m.Migrations {
		if ok, err := m.IsValid(migration.Version(), currentVersion); ok && err == nil {
			migrations = append(migrations, migration)
		}
	}

	return
}

func (m *Migrator) Run(currentVersion string) error {
	for _, migration := range m.GetValidMigrations(currentVersion) {
		err := migration.Migrate()
		if err != nil {
			return err
		}
	}

	return nil
}

// IsValid checks if versions constraints are met, assuming that currentVersion
// is just updated version and it should be taken for migration
func (m Migrator) IsValid(migrationVersion, currentVersion string) (bool, error) {

	// clean possible v prefixes
	migrationVersion = strings.TrimPrefix(migrationVersion, "v")
	currentVersion = strings.TrimPrefix(currentVersion, "v")

	if migrationVersion == "" || currentVersion == "" {
		return false, fmt.Errorf("empty version migration:'%s', current:'%s'", migrationVersion, currentVersion)
	}

	return version.Lte(currentVersion, migrationVersion)
}

func (m Migrator) ExecuteCommands(commands []string) (outputs []string, err error) {
	for _, command := range commands {
		out, err := process.ExecuteString(command)
		if err != nil {
			return outputs, err
		}

		outputs = append(outputs, string(out))
	}

	return outputs, nil
}
