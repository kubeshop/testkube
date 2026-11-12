package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	insightsv1 "github.com/kubeshop/testkube/api/insights/v1alpha1"
	"github.com/kubeshop/testkube/pkg/controlplane/declarative"
)

type recorded struct {
	method, path, auth string
	body               map[string]any
}

func serve(t *testing.T, status int, response string) (Client, *recorded) {
	t.Helper()
	rec := &recorded{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.method, rec.path, rec.auth = r.Method, r.URL.EscapedPath(), r.Header.Get("Authorization")
		if data, _ := io.ReadAll(r.Body); len(data) > 0 {
			require.NoError(t, json.Unmarshal(data, &rec.body))
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(response))
	}))
	t.Cleanup(srv.Close)
	return New(Config{URL: srv.URL + "/", Token: "user-token", OrganizationID: "org", EnvironmentID: "env", HTTPClient: srv.Client()}), rec
}

func boardKind(t *testing.T) declarative.Kind {
	kind, ok := declarative.LookupName("board")
	require.True(t, ok)
	return kind
}

func TestApply_SendsPreparedObjectToOrganizationScope(t *testing.T) {
	c, rec := serve(t, http.StatusOK, `{"metadata":{"name":"b"},"spec":{"displayName":"B"},"status":{"id":"uuid","creationSource":"api"}}`)
	board := &insightsv1.Board{
		ObjectMeta: metav1.ObjectMeta{Name: "b", Namespace: "ns", ResourceVersion: "3"},
		Spec:       insightsv1.BoardSpec{DisplayName: "B"},
		Status:     commonv1.ControlPlaneResourceStatus{GitOpsOwner: &commonv1.GitOpsOwner{AgentID: "forged"}},
	}

	stored, err := c.Apply(context.Background(), board)
	require.NoError(t, err)

	assert.Equal(t, http.MethodPut, rec.method)
	assert.Equal(t, "/organizations/org/resources/boards/b", rec.path, "boards are organization-scoped")
	assert.Equal(t, "Bearer user-token", rec.auth)
	assert.Equal(t, "insights.testkube.io/v1alpha1", rec.body["apiVersion"])
	assert.Equal(t, "Board", rec.body["kind"])
	assert.Equal(t, map[string]any{"name": "b"}, rec.body["metadata"])
	assert.NotContains(t, rec.body, "status", "ownership is never sent")

	out := stored.(*insightsv1.Board)
	assert.Equal(t, "uuid", out.Status.ID)
	assert.Equal(t, insightsv1.GroupVersion.WithKind(insightsv1.BoardKind), out.GroupVersionKind())
}

func TestGetAndList(t *testing.T) {
	c, rec := serve(t, http.StatusOK, `{"elements":[{"metadata":{"name":"a"}},{"metadata":{"name":"b"}}]}`)
	items, err := c.List(context.Background(), boardKind(t))
	require.NoError(t, err)
	assert.Equal(t, "/organizations/org/resources/boards", rec.path)
	require.Len(t, items, 2)
	assert.Equal(t, "b", items[1].GetName())

	c, rec = serve(t, http.StatusOK, `{"metadata":{"name":"a b"},"spec":{}}`)
	obj, err := c.Get(context.Background(), boardKind(t), "a b")
	require.NoError(t, err)
	assert.Equal(t, "/organizations/org/resources/boards/a%20b", rec.path)
	assert.Equal(t, "a b", obj.GetName())
}

func TestSendsTheConfiguredHeaders(t *testing.T) {
	var got http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Clone()
		_, _ = w.Write([]byte(`{"elements":[]}`))
	}))
	t.Cleanup(srv.Close)
	c := New(Config{
		URL: srv.URL, Token: "user-token", OrganizationID: "org", EnvironmentID: "env", HTTPClient: srv.Client(),
		Headers: map[string]string{"X-Proxy-Auth": "secret", "Authorization": "Bearer forged"},
	})

	_, err := c.List(context.Background(), boardKind(t))
	require.NoError(t, err)
	assert.Equal(t, "secret", got.Get("X-Proxy-Auth"), "a proxy's header reaches it")
	assert.Equal(t, "Bearer user-token", got.Get("Authorization"), "a header cannot replace the credential")
}

func TestDelete(t *testing.T) {
	c, rec := serve(t, http.StatusNoContent, "")
	require.NoError(t, c.Delete(context.Background(), boardKind(t), "b"))
	assert.Equal(t, http.MethodDelete, rec.method)
	assert.Equal(t, "/organizations/org/resources/boards/b", rec.path)
}

func TestErrors(t *testing.T) {
	board := &insightsv1.Board{ObjectMeta: metav1.ObjectMeta{Name: "b"}}

	t.Run("get of a missing resource", func(t *testing.T) {
		c, _ := serve(t, http.StatusNotFound, `{"title":"Not Found","detail":"board b not found"}`)
		_, err := c.Get(context.Background(), boardKind(t), "b")
		assert.ErrorIs(t, err, ErrNotFound)
		assert.Contains(t, err.Error(), "board b not found")
	})
	t.Run("apply to a Control Plane without the endpoints", func(t *testing.T) {
		c, _ := serve(t, http.StatusNotFound, "404 page not found")
		_, err := c.Apply(context.Background(), board)
		assert.ErrorIs(t, err, ErrUnsupported)
	})
	t.Run("list from a Control Plane without the endpoints", func(t *testing.T) {
		c, _ := serve(t, http.StatusNotFound, "")
		_, err := c.List(context.Background(), boardKind(t))
		assert.ErrorIs(t, err, ErrUnsupported)
	})
	t.Run("resource owned by a GitOps agent", func(t *testing.T) {
		c, _ := serve(t, http.StatusConflict, `{"detail":"managed by GitOps agent prod-sync"}`)
		_, err := c.Apply(context.Background(), board)
		assert.ErrorIs(t, err, ErrConflict)
		assert.Contains(t, err.Error(), "managed by GitOps agent prod-sync")
	})
	t.Run("unresolved references", func(t *testing.T) {
		c, _ := serve(t, http.StatusUnprocessableEntity, `{"detail":"unresolved references","errors":["mcpServers[0]: github not found",{"field":"members[1]","message":"no user x@y.z"}]}`)
		_, err := c.Apply(context.Background(), board)
		assert.ErrorIs(t, err, ErrInvalid)
		var statusErr *StatusError
		require.ErrorAs(t, err, &statusErr)
		assert.Equal(t, []string{"mcpServers[0]: github not found", "members[1]: no user x@y.z"}, statusErr.Errors)
	})
}

func TestRefusesAPITokenForOrganizationScope(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	t.Cleanup(srv.Close)
	c := New(Config{URL: srv.URL, Token: "tkcapi_abc", OrganizationID: "org", EnvironmentID: "env"})

	_, err := c.List(context.Background(), boardKind(t))
	assert.ErrorIs(t, err, ErrRequiresUserSession)
	assert.False(t, called, "nothing is sent")
}

func TestRequiresOrganization(t *testing.T) {
	c := New(Config{URL: "http://unused", Token: "t"})
	_, err := c.Get(context.Background(), boardKind(t), "b")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no organization")
}
