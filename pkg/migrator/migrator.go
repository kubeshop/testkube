package migrator

import (
	"fmt"
	"strings"

	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/log"
	"github.com/kubeshop/testkube/pkg/process"
	"github.com/kubeshop/testkube/pkg/semver"
)

type Migration interface {
	Migrate() error
	Version() string
	Info() string
	Type() MigrationType
}

// MigrationType is migration type
type MigrationType int

const (
	// MigrationTypeClient is client migration type
	MigrationTypeClient MigrationType = iota + 1
	// MigrationTypeServer is server migration type
	MigrationTypeServer
)

// NewMigrator returns new Migrator instance
func NewMigrator() *Migrator {
	return &Migrator{
		Log: log.DefaultLogger,
	}
}

// Migrator struct to manage migrations of Testkube API and CRDs
type Migrator struct {
	Migrations []Migration
	Log        *zap.SugaredLogger
}

// Add adds new migration
func (m *Migrator) Add(migration Migration) {
	m.Migrations = append(m.Migrations, migration)
}

// GetValidMigrations returns valid migration list for currentVersion
func (m *Migrator) GetValidMigrations(currentVersion string, migrationTypes ...MigrationType) (migrations []Migration) {
	types := make(map[MigrationType]struct{}, len(migrationTypes))
	for _, migrationType := range migrationTypes {
		types[migrationType] = struct{}{}
	}

	for _, migration := range m.Migrations {
		if ok, err := m.IsValid(migration.Version(), currentVersion); ok && err == nil {
			if _, ok = types[migration.Type()]; ok {
				migrations = append(migrations, migration)
			}
		}
	}

	return
}

// Run runs migrations of passed migration types
func (m *Migrator) Run(currentVersion string, migrationTypes ...MigrationType) error {
	for _, migration := range m.GetValidMigrations(currentVersion, migrationTypes...) {
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

	return semver.Lte(currentVersion, migrationVersion)
}

// ExecuteCommands executes multiple commands returns multiple commands outputs
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
