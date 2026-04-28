package mongo

import (
	"testing"

	sequencemongo "github.com/kubeshop/testkube/pkg/repository/sequence/mongo"
	testmongo "github.com/kubeshop/testkube/pkg/test/mongo"
	"github.com/kubeshop/testkube/pkg/utils/test"

	"github.com/kubeshop/testkube/pkg/repository/testworkflow/testsuite"
)

func TestMongoRepositorySuite_Integration(t *testing.T) {
	test.IntegrationTest(t)
	db, _ := testmongo.PrepareMongoTestDatabase(t, "repo_suite")

	seqRepo := sequencemongo.NewMongoRepository(db)
	repo := NewMongoRepository(db, false, WithMongoRepositorySequence(seqRepo))

	testsuite.RunRepositoryTests(t, repo)
}
