package deprecatedv1

import (
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	testsv3 "github.com/kubeshop/testkube-operator/api/tests/v3"
	testsclientv3 "github.com/kubeshop/testkube-operator/pkg/client/tests/v3"
	"github.com/kubeshop/testkube/cmd/api-server/commons"
	"github.com/kubeshop/testkube/pkg/log"
)

func TestDeprecatedTestkubeAPI_DeleteTest(t *testing.T) {
	app := fiber.New()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockDeprecatedClients := commons.NewMockDeprecatedClients(mockCtrl)
	mockDeprecatedClients.EXPECT().Tests().Return(getMockTestClient()).AnyTimes()

	s := &DeprecatedTestkubeAPI{
		Log:               log.DefaultLogger,
		DeprecatedClients: mockDeprecatedClients,
	}

	app.Delete("/tests/:id", s.DeleteTestHandler())

	req := httptest.NewRequest("DELETE", "http://localhost/tests/k6?skipDeleteExecutions=true", nil)
	resp, err := app.Test(req, -1)

	assert.NoError(t, err)
	defer resp.Body.Close()

}

func getMockTestClient() *testsclientv3.TestsClient {
	scheme := runtime.NewScheme()
	testsv3.AddToScheme(scheme)
	corev1.AddToScheme(scheme)
	initObjects := []k8sclient.Object{
		&testsv3.Test{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Test",
				APIVersion: "tests.testkube.io/v3",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "k6",
				Namespace: "",
			},
			Spec: testsv3.TestSpec{},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(initObjects...).
		Build()

	return testsclientv3.NewClient(fakeClient, "")
}
