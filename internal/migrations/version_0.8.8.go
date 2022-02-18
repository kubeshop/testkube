package migrations

import "github.com/kubeshop/testkube/pkg/migrator"

// add migration to global migrator
func init() {
	Migrator.Add(NewVersion_0_8_8())
}

func NewVersion_0_8_8() *Version_0_8_8 {
	return &Version_0_8_8{}
}

type Version_0_8_8 struct {
}

func (m *Version_0_8_8) Version() string {
	return "0.8.8"
}
func (m *Version_0_8_8) Migrate() error {
	commands := []string{
		`kubectl annotate --overwrite crds executors.executor.testkube.io meta.helm.sh/release-name=testkube meta.helm.sh/release-namespace=testkube`,
		`kubectl annotate --overwrite crds tests.tests.testkube.io meta.helm.sh/release-name=testkube meta.helm.sh/release-namespace=testkube`,
		`kubectl annotate --overwrite crds scripts.tests.testkube.io meta.helm.sh/release-name=testkube meta.helm.sh/release-namespace=testkube`,
		`kubectl label --overwrite crds executors.executor.testkube.io app.kubernetes.io/managed-by=Helm`,
		`kubectl label --overwrite crds tests.tests.testkube.io app.kubernetes.io/managed-by=Helm`,
		`kubectl label --overwrite crds scripts.tests.testkube.io app.kubernetes.io/managed-by=Helm`,
	}

	_, err := Migrator.ExecuteCommands(commands)
	return err
}
func (m *Version_0_8_8) Info() string {
	return "Adding labels and annotations to Testkube CRDs"
}

func (m *Version_0_8_8) Type() migrator.MigrationType {
	return migrator.MigrationTypeClient
}
