package migrator

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var ErrMigrationFailed = fmt.Errorf("migration failed")

func TestMigrator(t *testing.T) {

	t.Run("migrate versions one after another", func(t *testing.T) {
		// given
		migrator := NewMigrator()
		migrator.Add(&Migr1{})
		migrator.Add(&Migr2{})
		migrator.Add(&Migr3{})

		// when
		migrator.Run("0.0.2")

		fmt.Printf("%+v\n", migrator.Migrations[0])
		fmt.Printf("%+v\n", migrator.Migrations[1])
		fmt.Printf("%+v\n", migrator.Migrations[2])

		// then
		assert.Equal(t, migrator.Migrations[0].(*Migr1).Run, false)
		assert.Equal(t, migrator.Migrations[1].(*Migr2).Run, true)
		assert.Equal(t, migrator.Migrations[2].(*Migr3).Run, true)
	})

	t.Run("migrate mixed versions", func(t *testing.T) {
		// given
		migrator := NewMigrator()
		migrator.Add(&Migr3{})
		migrator.Add(&Migr1{})
		migrator.Add(&Migr2{})
		migrator.Add(&Migr1{})

		// when
		migrator.Run("0.0.2")

		// then
		assert.Equal(t, migrator.Migrations[0].(*Migr3).Run, true)
		assert.Equal(t, migrator.Migrations[1].(*Migr1).Run, false)
		assert.Equal(t, migrator.Migrations[2].(*Migr2).Run, true)
		assert.Equal(t, migrator.Migrations[3].(*Migr1).Run, false)
	})

	t.Run("failed migration returns error", func(t *testing.T) {
		// given
		migrator := NewMigrator()
		migrator.Add(&Migr1{})
		migrator.Add(&MigrFailed{})
		migrator.Add(&Migr1{})

		// when
		err := migrator.Run("0.0.1")

		// then
		assert.Error(t, err, ErrMigrationFailed)
	})

}

type Migr1 struct {
	Run bool
}

func (m *Migr1) Version() string {
	return "0.0.1"
}
func (m *Migr1) Migrate() error {
	m.Run = true
	return nil
}
func (m *Migr1) Info() string {
	return "some migration description 1"
}

type Migr2 struct {
	Run bool
}

func (m *Migr2) Version() string {
	return "0.0.2"
}
func (m *Migr2) Migrate() error {
	m.Run = true
	return nil
}
func (m *Migr2) Info() string {
	return "some migration description 2"
}

type Migr3 struct {
	Run bool
}

func (m *Migr3) Version() string {
	return "0.0.3"
}
func (m *Migr3) Migrate() error {
	m.Run = true
	return nil
}
func (m *Migr3) Info() string {
	return "some migration description 3"
}

type MigrFailed struct {
	Run bool
}

func (m *MigrFailed) Version() string {
	return "0.0.1"
}
func (m *MigrFailed) Migrate() error {
	m.Run = true
	return ErrMigrationFailed
}
func (m *MigrFailed) Info() string {
	return "some failed migration"
}
