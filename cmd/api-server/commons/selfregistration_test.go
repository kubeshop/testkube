package commons

import (
	"context"
	"errors"
	"net"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/secretmanager"
)

type fakeSecretWriter struct {
	createErrs []error // returned in order; nil once exhausted
	updateErrs []error
	creates    int
	updates    int
	lastData   map[string]string
}

func (f *fakeSecretWriter) Create(_ context.Context, _, _ string, data map[string]string, _ secretmanager.CreateOptions) (testkube.Secret, error) {
	f.creates++
	f.lastData = data
	return testkube.Secret{}, next(&f.createErrs)
}

func (f *fakeSecretWriter) Update(_ context.Context, _, _ string, data map[string]string, _ secretmanager.UpdateOptions) (testkube.Secret, error) {
	f.updates++
	f.lastData = data
	return testkube.Secret{}, next(&f.updateErrs)
}

func next(errs *[]error) error {
	if len(*errs) == 0 {
		return nil
	}
	err := (*errs)[0]
	*errs = (*errs)[1:]
	return err
}

type nopLogger struct{ warns int }

func (l *nopLogger) Warnw(string, ...interface{}) { l.warns++ }
func (l *nopLogger) Infow(string, ...interface{}) {}

var (
	gr           = schema.GroupResource{Resource: "secrets"}
	errConnReset = &net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET} // "read: connection reset by peer"
	errExists    = k8serrors.NewAlreadyExists(gr, "self-registration")
	errForbidden = k8serrors.NewForbidden(gr, "self-registration", errors.New("rbac"))
	testBackoff  = []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond}
	testData     = map[string]string{"TESTKUBE_PRO_API_KEY": "key", "TESTKUBE_PRO_AGENT_ID": "id"}
	maxTestCalls = len(testBackoff) + 1
)

func TestPersistSelfRegistrationSecret(t *testing.T) {
	tests := []struct {
		name        string
		createErrs  []error
		updateErrs  []error
		wantErr     bool
		wantCreates int
		wantUpdates int
		wantWarns   int
	}{
		{name: "first try succeeds", wantCreates: 1},
		{name: "transient error then success", createErrs: []error{errConnReset, errConnReset}, wantCreates: 3, wantWarns: 2},
		{name: "already exists is updated", createErrs: []error{errExists}, wantCreates: 1, wantUpdates: 1},
		{name: "already exists, update fails transiently then succeeds",
			createErrs: []error{errExists, errExists}, updateErrs: []error{errConnReset}, wantCreates: 2, wantUpdates: 2, wantWarns: 1},
		{name: "gives up after bounded attempts", createErrs: []error{errConnReset, errConnReset, errConnReset, errConnReset, errConnReset},
			wantErr: true, wantCreates: maxTestCalls, wantWarns: maxTestCalls - 1},
		{name: "forbidden is not retried", createErrs: []error{errForbidden}, wantErr: true, wantCreates: 1},
		{name: "creation disabled is not retried", createErrs: []error{secretmanager.ErrCreateDisabled}, wantErr: true, wantCreates: 1},
		{name: "existing secret not owned by testkube is not retried",
			createErrs: []error{errExists}, updateErrs: []error{secretmanager.ErrNotControlled}, wantErr: true, wantCreates: 1, wantUpdates: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &fakeSecretWriter{createErrs: tt.createErrs, updateErrs: tt.updateErrs}
			l := &nopLogger{}
			err := PersistSelfRegistrationSecret(context.Background(), l, sm, "ns", "self-registration", testData, testBackoff)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "self-registration")
			} else {
				require.NoError(t, err)
				assert.Equal(t, testData, sm.lastData)
			}
			assert.Equal(t, tt.wantCreates, sm.creates, "creates")
			assert.Equal(t, tt.wantUpdates, sm.updates, "updates")
			assert.Equal(t, tt.wantWarns, l.warns, "retry warnings")
		})
	}
}

func TestPersistSelfRegistrationSecret_StopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sm := &fakeSecretWriter{createErrs: []error{errConnReset, errConnReset}}
	err := PersistSelfRegistrationSecret(ctx, &nopLogger{}, sm, "ns", "self-registration", testData, []time.Duration{time.Hour, time.Hour})
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, sm.creates)
}
