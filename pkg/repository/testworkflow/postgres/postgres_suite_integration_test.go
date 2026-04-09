package postgres

import (
	"testing"

	testpostgres "github.com/kubeshop/testkube/pkg/test/postgres"
	"github.com/kubeshop/testkube/pkg/utils/test"

	"github.com/kubeshop/testkube/pkg/repository/testworkflow/testsuite"
)

func TestPostgresRepositorySuite_Integration(t *testing.T) {
	test.IntegrationTest(t)
	testDB, _ := testpostgres.PreparePostgresTestDatabase(t, "repo_suite")

	repo := NewPostgresRepository(
		testDB.Pool,
		WithOrganizationID("test-org"),
		WithEnvironmentID("test-env"),
	)

	testsuite.RunRepositoryTests(t, repo)
}
