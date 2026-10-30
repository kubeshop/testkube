package controlplane

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/cloud"
	"github.com/kubeshop/testkube/pkg/identitytoken"
	executionv1 "github.com/kubeshop/testkube/pkg/proto/testkube/testworkflow/execution/v1"
	"github.com/kubeshop/testkube/pkg/repository/testworkflow"
)

func newTestIdentitySigner(t *testing.T) *identitytoken.Signer {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)
	signer, err := identitytoken.NewSigner("https://testkube.example.com", key)
	require.NoError(t, err)
	return signer
}

type recordingSigner struct {
	*identitytoken.Signer
	subject  identitytoken.Subject
	audience string
	err      error
}

func (r *recordingSigner) Sign(subject identitytoken.Subject, audience string) (string, error) {
	r.subject, r.audience = subject, audience
	if r.err != nil {
		return "", r.err
	}
	return "signed-token", nil
}

func withAPIKey(token string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs(apiKeyMetadata, token))
}

func TestGetIdentityToken(t *testing.T) {
	base := newTestIdentitySigner(t)
	validToken := base.ExecutionToken(common.StandaloneEnvironment, "exec-1")

	running := testkube.TestWorkflowExecution{
		Id:       "exec-1",
		RunnerId: common.StandaloneRunner,
		Workflow: &testkube.TestWorkflow{Name: "deploy"},
		Result:   &testkube.TestWorkflowResult{Status: common.Ptr(testkube.RUNNING_TestWorkflowStatus)},
	}
	finished := running
	finished.Result = &testkube.TestWorkflowResult{Status: common.Ptr(testkube.PASSED_TestWorkflowStatus), FinishedAt: time.Now()}
	noStatus := running
	noStatus.Result = &testkube.TestWorkflowResult{}
	noWorkflow := running
	noWorkflow.Workflow = nil
	// Get also matches by name, so "exec-1" can resolve to another execution named exec-1.
	matchedByName := running
	matchedByName.Id = "exec-other"

	tests := []struct {
		name      string
		noSigner  bool
		ctx       context.Context
		req       *cloud.GetIdentityTokenRequest
		execution *testkube.TestWorkflowExecution
		lookupErr error
		signErr   error
		wantCode  codes.Code
	}{
		{name: "mints a token for the running execution", ctx: withAPIKey(validToken), req: &cloud.GetIdentityTokenRequest{ExecutionId: "exec-1", Audience: "sts.amazonaws.com"}, execution: &running, wantCode: codes.OK},
		{name: "result without status counts as running", ctx: withAPIKey(validToken), req: &cloud.GetIdentityTokenRequest{ExecutionId: "exec-1", Audience: "aud"}, execution: &noStatus, wantCode: codes.OK},
		{name: "not configured", noSigner: true, ctx: withAPIKey(validToken), req: &cloud.GetIdentityTokenRequest{ExecutionId: "exec-1", Audience: "aud"}, wantCode: codes.FailedPrecondition},
		{name: "missing execution ID", ctx: withAPIKey(validToken), req: &cloud.GetIdentityTokenRequest{Audience: "aud"}, wantCode: codes.InvalidArgument},
		{name: "no execution token", ctx: context.Background(), req: &cloud.GetIdentityTokenRequest{ExecutionId: "exec-1", Audience: "aud"}, wantCode: codes.PermissionDenied},
		{name: "another execution's token", ctx: withAPIKey(base.ExecutionToken(common.StandaloneEnvironment, "exec-2")), req: &cloud.GetIdentityTokenRequest{ExecutionId: "exec-1", Audience: "aud"}, wantCode: codes.PermissionDenied},
		{name: "unknown execution", ctx: withAPIKey(validToken), req: &cloud.GetIdentityTokenRequest{ExecutionId: "exec-1", Audience: "aud"}, lookupErr: errors.New("not found"), wantCode: codes.Unknown},
		{name: "finished execution", ctx: withAPIKey(validToken), req: &cloud.GetIdentityTokenRequest{ExecutionId: "exec-1", Audience: "aud"}, execution: &finished, wantCode: codes.NotFound},
		{name: "execution matched by name only", ctx: withAPIKey(validToken), req: &cloud.GetIdentityTokenRequest{ExecutionId: "exec-1", Audience: "aud"}, execution: &matchedByName, wantCode: codes.NotFound},
		{name: "execution without workflow", ctx: withAPIKey(validToken), req: &cloud.GetIdentityTokenRequest{ExecutionId: "exec-1", Audience: "aud"}, execution: &noWorkflow, wantCode: codes.FailedPrecondition},
		{name: "invalid audience", ctx: withAPIKey(validToken), req: &cloud.GetIdentityTokenRequest{ExecutionId: "exec-1"}, execution: &running, signErr: identitytoken.ErrInvalidAudience, wantCode: codes.InvalidArgument},
		{name: "signing failure", ctx: withAPIKey(validToken), req: &cloud.GetIdentityTokenRequest{ExecutionId: "exec-1", Audience: "aud"}, execution: &running, signErr: errors.New("boom"), wantCode: codes.Internal},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			repo := testworkflow.NewMockRepository(ctrl)
			if tc.execution != nil || tc.lookupErr != nil {
				var execution testkube.TestWorkflowExecution
				if tc.execution != nil {
					execution = *tc.execution
				}
				repo.EXPECT().Get(gomock.Any(), tc.req.ExecutionId).Return(execution, tc.lookupErr)
			}
			signer := &recordingSigner{Signer: base, err: tc.signErr}
			cfg := Config{OrganizationID: "tkcorg_1"}
			if !tc.noSigner {
				cfg.IdentityTokens = signer
			}
			server := &Server{cfg: cfg, resultsRepository: repo, envID: "tkcenv_1"}

			resp, err := server.GetIdentityToken(tc.ctx, tc.req)

			require.Equal(t, tc.wantCode, status.Code(err), "error: %v", err)
			if tc.wantCode != codes.OK {
				assert.Nil(t, resp)
				return
			}
			assert.Equal(t, "signed-token", resp.Token)
			assert.Equal(t, identitytoken.Subject{
				OrganizationID: "tkcorg_1",
				EnvironmentID:  "tkcenv_1",
				WorkflowName:   "deploy",
				ExecutionID:    "exec-1",
				RunnerID:       common.StandaloneRunner,
			}, signer.subject)
			assert.Equal(t, tc.req.Audience, signer.audience)
		})
	}
}

// The token handed out on dispatch is the one GetIdentityToken accepts, so the
// pod built from it can mint identity tokens and nothing else can.
func TestGetExecutionUpdates_HandsOutTheExecutionToken(t *testing.T) {
	signer := newTestIdentitySigner(t)
	querier := &fakeExecutionQuerier{toStart: []testkube.TestWorkflowExecution{assignedExecution("exec-1"), assignedExecution("exec-2")}}
	server := newDispatchServer(t, querier, &fakeController{}, Config{IdentityTokens: signer})

	response, err := server.GetExecutionUpdates(context.Background(), &executionv1.GetExecutionUpdatesRequest{})

	require.NoError(t, err)
	require.Len(t, response.GetStart(), 2)
	for _, start := range response.GetStart() {
		token := start.GetExecutionToken()
		assert.True(t, signer.VerifyExecutionToken(start.GetEnvironmentId(), start.GetExecutionId(), token), start.GetExecutionId())
	}
	assert.NotEqual(t, response.GetStart()[0].GetExecutionToken(), response.GetStart()[1].GetExecutionToken())
}

// With identity tokens off the runner keeps using the agent key, as before.
func TestGetExecutionUpdates_NoExecutionTokenWithoutIdentityTokens(t *testing.T) {
	querier := &fakeExecutionQuerier{toStart: []testkube.TestWorkflowExecution{assignedExecution("exec-1")}}
	server := newDispatchServer(t, querier, &fakeController{}, Config{})

	response, err := server.GetExecutionUpdates(context.Background(), &executionv1.GetExecutionUpdatesRequest{})

	require.NoError(t, err)
	require.Len(t, response.GetStart(), 1)
	assert.Empty(t, response.GetStart()[0].GetExecutionToken())
}
