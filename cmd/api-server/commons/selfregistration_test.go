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
	// onCreate, when set, runs on each Create call (1-based) and may return an error that overrides createErrs.
	onCreate func(ctx context.Context, call int) error
}

func (f *fakeSecretWriter) Create(ctx context.Context, _, _ string, data map[string]string, _ secretmanager.CreateOptions) (testkube.Secret, error) {
	f.creates++
	f.lastData = data
	if f.onCreate != nil {
		if err := f.onCreate(ctx, f.creates); err != nil {
			return testkube.Secret{}, err
		}
		return testkube.Secret{}, nil
	}
	return testkube.Secret{}, popSecretErr(&f.createErrs)
}

func (f *fakeSecretWriter) Update(_ context.Context, _, _ string, data map[string]string, _ secretmanager.UpdateOptions) (testkube.Secret, error) {
	f.updates++
	f.lastData = data
	return testkube.Secret{}, popSecretErr(&f.updateErrs)
}

func popSecretErr(errs *[]error) error {
	if len(*errs) == 0 {
		return nil
	}
	err := (*errs)[0]
	*errs = (*errs)[1:]
	return err
}

type countingLogger struct{ warns, infos int }

func (l *countingLogger) Warnw(string, ...interface{}) { l.warns++ }
func (l *countingLogger) Infow(string, ...interface{}) { l.infos++ }

var (
	secretsGroupResource = schema.GroupResource{Resource: "secrets"}
	errSecretConnReset   = &net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET} // "read: connection reset by peer"
	errSecretExists      = k8serrors.NewAlreadyExists(secretsGroupResource, "self-registration")
	errSecretForbidden   = k8serrors.NewForbidden(secretsGroupResource, "self-registration", errors.New("rbac"))
	testSecretBackoff    = []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond}
	testSecretData       = map[string]string{"TESTKUBE_PRO_API_KEY": "key", "TESTKUBE_PRO_AGENT_ID": "id"}
	maxTestSecretCalls   = len(testSecretBackoff) + 1
)

func TestCheckSelfRegistrationSecretWritable(t *testing.T) {
	tests := []struct {
		name        string
		createErrs  []error
		updateErrs  []error
		wantErr     func(error) bool
		wantCreates int
		wantUpdates int
	}{
		{name: "creates an empty secret, then checks update", wantCreates: 1, wantUpdates: 1},
		{name: "update forbidden after create fails before registration",
			updateErrs: []error{errSecretForbidden}, wantErr: k8serrors.IsForbidden, wantCreates: 1, wantUpdates: 1},
		{name: "existing managed secret is updated without data", createErrs: []error{errSecretExists}, wantCreates: 1, wantUpdates: 1},
		{name: "transient error is retried", createErrs: []error{errSecretConnReset}, wantCreates: 2, wantUpdates: 1},
		{name: "create forbidden fails before registration", createErrs: []error{errSecretForbidden}, wantErr: k8serrors.IsForbidden, wantCreates: 1},
		{name: "existing unmanaged secret fails before registration",
			createErrs: []error{errSecretExists}, updateErrs: []error{secretmanager.ErrNotFound},
			wantErr: func(err error) bool { return errors.Is(err, errSelfRegistrationSecretNotManaged) }, wantCreates: 1, wantUpdates: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &fakeSecretWriter{createErrs: tt.createErrs, updateErrs: tt.updateErrs, lastData: testSecretData}
			err := CheckSelfRegistrationSecretWritable(context.Background(), &countingLogger{}, sm, "ns", "self-registration",
				SelfRegistrationOptions{Backoff: testSecretBackoff})
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.True(t, tt.wantErr(err), "unexpected error: %v", err)
			} else {
				require.NoError(t, err)
			}
			// nil data: Create makes an empty secret and Update keeps the existing data.
			assert.Nil(t, sm.lastData)
			assert.Equal(t, tt.wantCreates, sm.creates, "creates")
			assert.Equal(t, tt.wantUpdates, sm.updates, "updates")
		})
	}
}

func TestPersistSelfRegistrationSecret(t *testing.T) {
	tests := []struct {
		name        string
		createErrs  []error
		updateErrs  []error
		wantErr     bool
		wantCreates int
		wantUpdates int
		wantWarns   int
		wantInfos   int
		wantMsg     string
	}{
		{name: "first try succeeds", wantCreates: 1},
		{name: "transient error then success", createErrs: []error{errSecretConnReset, errSecretConnReset}, wantCreates: 3, wantWarns: 2, wantInfos: 1},
		{name: "already exists is updated", createErrs: []error{errSecretExists}, wantCreates: 1, wantUpdates: 1},
		{name: "already exists, update fails transiently then succeeds",
			createErrs: []error{errSecretExists, errSecretExists}, updateErrs: []error{errSecretConnReset}, wantCreates: 2, wantUpdates: 2, wantWarns: 1, wantInfos: 1},
		{name: "gives up after bounded attempts", createErrs: []error{errSecretConnReset, errSecretConnReset, errSecretConnReset, errSecretConnReset, errSecretConnReset},
			wantErr: true, wantCreates: maxTestSecretCalls, wantWarns: maxTestSecretCalls - 1},
		{name: "forbidden is not retried", createErrs: []error{errSecretForbidden}, wantErr: true, wantCreates: 1},
		{name: "creation disabled is not retried", createErrs: []error{secretmanager.ErrCreateDisabled}, wantErr: true, wantCreates: 1},
		{name: "existing secret not owned by testkube is not retried",
			createErrs: []error{errSecretExists}, updateErrs: []error{secretmanager.ErrNotControlled}, wantErr: true, wantCreates: 1, wantUpdates: 1,
			wantMsg: "missing label createdBy=testkube"},
		{name: "existing unlabelled secret (update reports not found) is not retried",
			createErrs: []error{errSecretExists}, updateErrs: []error{secretmanager.ErrNotFound}, wantErr: true, wantCreates: 1, wantUpdates: 1,
			wantMsg: `secret "runner-self-registration" exists but is not visible to Testkube`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := &fakeSecretWriter{createErrs: tt.createErrs, updateErrs: tt.updateErrs}
			l := &countingLogger{}
			err := PersistSelfRegistrationSecret(context.Background(), l, sm, "ns", "self-registration", testSecretData, SelfRegistrationOptions{Backoff: testSecretBackoff, SecretPrefix: "runner-"})
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "self-registration")
				if tt.wantMsg != "" {
					assert.Contains(t, err.Error(), tt.wantMsg)
					assert.ErrorIs(t, err, errSelfRegistrationSecretNotManaged)
				}
			} else {
				require.NoError(t, err)
				assert.Equal(t, testSecretData, sm.lastData)
			}
			assert.Equal(t, tt.wantCreates, sm.creates, "creates")
			assert.Equal(t, tt.wantUpdates, sm.updates, "updates")
			assert.Equal(t, tt.wantWarns, l.warns, "retry warnings")
			assert.Equal(t, tt.wantInfos, l.infos, "saved-after-retry logs")
		})
	}
}

func TestPersistSelfRegistrationSecret_StopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sm := &fakeSecretWriter{createErrs: []error{errSecretConnReset, errSecretConnReset}}
	err := PersistSelfRegistrationSecret(ctx, &countingLogger{}, sm, "ns", "self-registration", testSecretData, SelfRegistrationOptions{Backoff: []time.Duration{time.Hour, time.Hour}})
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, sm.creates)
}

func TestPersistSelfRegistrationSecret_CancelDuringBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sm := &fakeSecretWriter{onCreate: func(context.Context, int) error {
		go func() { time.Sleep(10 * time.Millisecond); cancel() }()
		return errSecretConnReset
	}}
	err := PersistSelfRegistrationSecret(ctx, &countingLogger{}, sm, "ns", "self-registration", testSecretData,
		SelfRegistrationOptions{Backoff: []time.Duration{time.Hour, time.Hour}})
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 1, sm.creates)
}

func TestPersistSelfRegistrationSecret_PerAttemptTimeoutIsRetried(t *testing.T) {
	sm := &fakeSecretWriter{onCreate: func(ctx context.Context, call int) error {
		if call == 1 {
			<-ctx.Done() // hung API call: only the per-attempt timeout ends it
			return ctx.Err()
		}
		return nil
	}}
	l := &countingLogger{}
	err := PersistSelfRegistrationSecret(context.Background(), l, sm, "ns", "self-registration", testSecretData,
		SelfRegistrationOptions{Backoff: testSecretBackoff, WriteTimeout: 20 * time.Millisecond})
	require.NoError(t, err)
	assert.Equal(t, 2, sm.creates)
	assert.Equal(t, 1, l.warns)
	assert.Equal(t, 1, l.infos)
}

func TestDefaultSelfRegistrationSecretBackoff_ReturnsFreshSlice(t *testing.T) {
	first := DefaultSelfRegistrationSecretBackoff()
	first[0] = time.Hour
	assert.Equal(t, selfRegistrationBackoffFirst, DefaultSelfRegistrationSecretBackoff()[0])
}
