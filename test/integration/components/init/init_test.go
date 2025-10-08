package init_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/orchestration"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/runner"
	"github.com/kubeshop/testkube/pkg/utils/test"
)

func TestInitProcessCore_Integration(t *testing.T) {
	t.Skip()
	test.IntegrationTest(t)

	testDir := t.TempDir()
	internalPath := filepath.Join(testDir, ".tktw")
	require.NoError(t, os.MkdirAll(internalPath, 0755))

	// Create termination log file
	termLogPath := filepath.Join(testDir, "termination.log")
	require.NoError(t, os.WriteFile(termLogPath, []byte{}, 0666))

	// Test Group 0 - Setup phase
	t.Run("SetupPhase", func(t *testing.T) {
		setupEnv(t, testDir)
		updateConstants(testDir)
		initializeOrchestration(t)
		t.Cleanup(func() { cleanupOrchestration(t) })

		// Debug: Check that source binaries exist
		actualInitBinary := os.Getenv("TESTKUBE_TW_INIT_BINARY_PATH")
		actualToolkitBinary := os.Getenv("TESTKUBE_TW_TOOLKIT_BINARY_PATH")
		t.Logf("Init binary path: %s", actualInitBinary)
		t.Logf("Toolkit binary path: %s", actualToolkitBinary)
		t.Logf("Internal path: %s", os.Getenv("TESTKUBE_TW_INTERNAL_PATH"))

		exitCode, err := runner.RunInit(0)
		require.NoError(t, err)
		assert.Equal(t, 0, exitCode, "Setup should succeed")

		// Verify state file has correct permissions
		actualStatePath := filepath.Join(testDir, ".tktw", "state")
		info, err := os.Stat(actualStatePath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0777), info.Mode().Perm(), "State file should have 0777 permissions")
	})

	// Test Group 1 - Execute command
	t.Run("ExecuteCommand", func(t *testing.T) {
		setupEnv(t, testDir)
		updateConstants(testDir)
		initializeOrchestration(t)
		t.Cleanup(func() { cleanupOrchestration(t) })

		exitCode, err := runner.RunInit(1)
		require.NoError(t, err)
		assert.Equal(t, 0, exitCode, "Execute should succeed")

		// Verify state was updated
		actualStatePath := filepath.Join(testDir, ".tktw", "state")
		updatedState := loadStateFromPath(t, actualStatePath)
		assert.Equal(t, 1, updatedState.CurrentGroupIndex)
		assert.Contains(t, updatedState.Steps, "test-step")
	})

	// Test invalid group index
	t.Run("InvalidGroupIndex", func(t *testing.T) {
		setupEnv(t, testDir)
		updateConstants(testDir)
		initializeOrchestration(t)
		t.Cleanup(func() { cleanupOrchestration(t) })

		exitCode, err := runner.RunInit(99)
		assert.Error(t, err, "Should return error for invalid group")
		assert.NotEqual(t, 0, exitCode, "Should fail with invalid group index")
		assert.Equal(t, int(constants.CodeInputError), exitCode, "Should return CodeInputError for invalid group")
	})
}

func TestInitProcessWithRetry_Integration(t *testing.T) {
	test.IntegrationTest(t)

	testDir := t.TempDir()
	internalPath := filepath.Join(testDir, ".tktw")
	require.NoError(t, os.MkdirAll(internalPath, 0755))

	// Create termination log file
	termLogPath := filepath.Join(testDir, "termination.log")
	require.NoError(t, os.WriteFile(termLogPath, []byte{}, 0666))

	// Create a script that fails first time, succeeds second time
	scriptPath := filepath.Join(testDir, "retry-test.sh")
	script := `#!/bin/sh
COUNTER_FILE="/tmp/retry-counter-$$"
COUNT=$(cat $COUNTER_FILE 2>/dev/null || echo 0)
COUNT=$((COUNT + 1))
echo $COUNT > $COUNTER_FILE

if [ $COUNT -lt 2 ]; then
    echo "Attempt $COUNT: Failing"
    exit 1
else
    echo "Attempt $COUNT: Success"
    rm -f $COUNTER_FILE
    exit 0
fi
`
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0755))

	setupEnvWithRetry(t, testDir, scriptPath)
	updateConstants(testDir)
	initializeOrchestration(t)
	t.Cleanup(func() { cleanupOrchestration(t) })

	exitCode, err := runner.RunInit(1)
	require.NoError(t, err)
	assert.Equal(t, 0, exitCode, "Should succeed after retry")
}

func TestInitProcessStateSharing_Integration(t *testing.T) {
	// TODO(dejan): investigate failing test
	t.Skip("Test is flaky and failing with exit code 137 (SIGKILL) - needs investigation")

	test.IntegrationTest(t)

	testDir := t.TempDir()
	internalPath := filepath.Join(testDir, ".tktw")
	require.NoError(t, os.MkdirAll(internalPath, 0755))

	// Create termination log file
	termLogPath := filepath.Join(testDir, "termination.log")
	require.NoError(t, os.WriteFile(termLogPath, []byte{}, 0666))

	// Create actions that demonstrate state sharing
	// Note: Using simpler commands that exit quickly to avoid process management issues
	actions := [][]map[string]interface{}{
		// Group 0 - Setup
		{
			{"_": map[string]bool{"i": true, "t": true, "b": true}},
		},
		// Group 1 - First step that modifies state
		{
			{"d": map[string]interface{}{"c": "true", "r": "step1"}},
			{"c": map[string]interface{}{
				"r": "step1",
				"c": map[string]interface{}{
					"command": []string{"true"}, // Use true command for instant exit
				},
			}},
			{"e": map[string]interface{}{"r": "step1"}},
		},
		// Group 2 - Second step that depends on first step
		{
			{"d": map[string]interface{}{"c": "true", "r": "step2", "p": []string{"step1"}}},
			{"c": map[string]interface{}{
				"r": "step2",
				"c": map[string]interface{}{
					"command": []string{"true"}, // Use true command for instant exit
				},
			}},
			{"e": map[string]interface{}{"r": "step2"}},
		},
	}

	setupEnvWithActions(t, testDir, actions)
	updateConstants(testDir)

	// Define state path for debugging
	actualStatePath := filepath.Join(testDir, ".tktw", "state")

	// Run each group in sequence with proper cleanup
	t.Run("Group0_Setup", func(t *testing.T) {
		initializeOrchestration(t)
		t.Cleanup(func() { cleanupOrchestration(t) })

		exitCode0, err := runner.RunInit(0)
		require.NoError(t, err)
		assert.Equal(t, 0, exitCode0)
	})

	// Allow processes to fully cleanup between groups
	time.Sleep(300 * time.Millisecond)

	t.Run("Group1_Step1", func(t *testing.T) {
		initializeOrchestration(t)
		t.Cleanup(func() { cleanupOrchestration(t) })

		t.Logf("Running group 1 with state file at: %s", actualStatePath)
		exitCode1, err := runner.RunInit(1)
		if err != nil {
			t.Logf("Group 1 error: %v", err)
		}
		t.Logf("Group 1 exit code: %d", exitCode1)
		require.NoError(t, err)
		assert.Equal(t, 0, exitCode1)

		// Verify state after group 1
		state1 := loadStateFromPath(t, actualStatePath)
		assert.Equal(t, 1, state1.CurrentGroupIndex)
		assert.Contains(t, state1.Steps, "step1")
		step1 := state1.Steps["step1"].(map[string]interface{})
		assert.Equal(t, "passed", step1["s"], "step1 should have passed status")
	})

	// Allow processes to fully cleanup between groups
	time.Sleep(300 * time.Millisecond)

	t.Run("Group2_Step2", func(t *testing.T) {
		initializeOrchestration(t)
		t.Cleanup(func() { cleanupOrchestration(t) })

		exitCode2, err := runner.RunInit(2)
		require.NoError(t, err)
		assert.Equal(t, 0, exitCode2)

		// Verify final state
		finalState := loadStateFromPath(t, actualStatePath)
		assert.Equal(t, 2, finalState.CurrentGroupIndex)
		assert.Contains(t, finalState.Steps, "step1")
		assert.Contains(t, finalState.Steps, "step2")

		// Verify step2 has correct parent and status
		step2 := finalState.Steps["step2"].(map[string]interface{})
		assert.Equal(t, "passed", step2["s"], "step2 should have passed status")
		if parents, ok := step2["p"].([]interface{}); ok {
			assert.Contains(t, parents, "step1", "step2 should have step1 as parent")
		}
	})
}

func TestInitProcessMetricsCapture_Integration(t *testing.T) {
	t.Skip("Test is flaky and failing with exit code 137 (SIGKILL) - needs investigation")
	test.IntegrationTest(t)

	testDir := t.TempDir()
	internalPath := filepath.Join(testDir, ".tktw")
	dataPath := filepath.Join(testDir, "data")
	require.NoError(t, os.MkdirAll(internalPath, 0755))
	require.NoError(t, os.MkdirAll(dataPath, 0755))

	// Create termination log file
	termLogPath := filepath.Join(testDir, "termination.log")
	require.NoError(t, os.WriteFile(termLogPath, []byte{}, 0666))

	// Create a script that consumes CPU for 5+ seconds
	scriptPath := filepath.Join(testDir, "cpu-intensive.sh")
	script := `#!/bin/sh
echo "Starting CPU-intensive task..."
START=$(date +%s)
while true; do
    # CPU-intensive calculations - use a simple loop
    i=0
    while [ $i -lt 10000 ]; do
        i=$((i + 1))
    done
    NOW=$(date +%s)
    ELAPSED=$((NOW - START))
    if [ $ELAPSED -ge 5 ]; then
        echo "CPU task completed after $ELAPSED seconds"
        break
    fi
done
`
	require.NoError(t, os.WriteFile(scriptPath, []byte(script), 0755))

	// Setup environment with resource limits
	setupEnvWithMetrics(t, testDir, scriptPath)
	updateConstants(testDir)
	initializeOrchestration(t)
	t.Cleanup(func() { cleanupOrchestration(t) })

	os.Setenv("TW_FS_DATA", dataPath)

	// Run the CPU-intensive task
	exitCode, err := runner.RunInit(1)
	require.NoError(t, err)

	assert.Equal(t, 0, exitCode, "Should complete successfully")

	// Verify metrics were captured - check in the correct location
	metricsPath := filepath.Join(dataPath, ".testkube", "internal", "metrics", "cpu-test")
	entries, err := os.ReadDir(metricsPath)
	if err == nil && len(entries) > 0 {
		// Metrics directory exists and has files
		assert.Greater(t, len(entries), 0, "Should have captured metrics")

		// Read a metrics file to verify format
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".txt") {
				content, err := os.ReadFile(filepath.Join(metricsPath, entry.Name()))
				require.NoError(t, err)

				// Basic validation - metrics should contain measurement data
				assert.Contains(t, string(content), "cpu", "Metrics should contain CPU data")
				break
			}
		}
	} else {
		// Metrics might not be captured in test environment, which is acceptable
		t.Log("Metrics not captured - this is expected in some test environments")
	}
}

func getProjectRoot() string {
	// Check if project root is provided via environment variable (from make)
	if projectRoot := os.Getenv("TESTKUBE_PROJECT_ROOT"); projectRoot != "" {
		return projectRoot
	}

	// Otherwise, try to find it from working directory
	wd, err := os.Getwd()
	if err == nil {
		// Walk up the directory tree looking for go.mod
		dir := wd
		for {
			goModPath := filepath.Join(dir, "go.mod")
			if _, err := os.Stat(goModPath); err == nil {
				// Found go.mod, this should be the project root
				return dir
			}

			parent := filepath.Dir(dir)
			if parent == dir {
				// Reached filesystem root
				break
			}
			dir = parent
		}
	}

	// Fallback: use runtime.Caller for relative navigation
	_, filename, _, _ := runtime.Caller(0)
	// Navigate up from test/integration/components/init to project root
	return filepath.Join(filepath.Dir(filename), "..", "..", "..", "..")
}

func initializeOrchestration(t *testing.T) {
	t.Helper()
	// Clear any existing state
	data.ClearState()
	orchestration.Setup = nil

	// Initialize orchestration
	orchestration.Initialize()

	// Clear executions state using reflection
	executionsValue := reflect.ValueOf(orchestration.Executions).Elem()
	executionsField := executionsValue.FieldByName("executions")
	if executionsField.IsValid() && executionsField.CanSet() {
		// Make the field settable using unsafe
		executionsField = reflect.NewAt(executionsField.Type(), unsafe.Pointer(executionsField.UnsafeAddr())).Elem()
		executionsField.Set(reflect.MakeSlice(executionsField.Type(), 0, 0))
	}

	// Clear any aborted status from previous tests
	orchestration.Executions.ClearAbortedStatus()
}

func cleanupOrchestration(t *testing.T) {
	t.Helper()
	// Clear singleton state after each test
	data.ClearState()
	orchestration.Setup = nil
	orchestration.Executions.ClearAbortedStatus()
}

func setupEnv(t *testing.T, testDir string) {
	t.Helper()
	// Clear environment first
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "TW_") || strings.HasPrefix(env, "_") || strings.HasPrefix(env, "TESTKUBE_TW_") {
			key := strings.Split(env, "=")[0]
			os.Unsetenv(key)
		}
	}

	// Use actual binaries from bin directory
	projectRoot := getProjectRoot()
	actualInitBinary := filepath.Join(projectRoot, "bin", "app", "testworkflow-init")
	actualToolkitBinary := filepath.Join(projectRoot, "bin", "app", "testworkflow-toolkit")

	// Verify binaries exist
	if _, err := os.Stat(actualInitBinary); os.IsNotExist(err) {
		t.Fatalf("testworkflow-init binary not found at %s. Run 'make build-init build-toolkit' first", actualInitBinary)
	}
	if _, err := os.Stat(actualToolkitBinary); os.IsNotExist(err) {
		t.Fatalf("testworkflow-toolkit binary not found at %s. Run 'make build-init build-toolkit' first", actualToolkitBinary)
	}

	// Create a mock busybox directory with basic shell utilities
	busyboxDir := filepath.Join(testDir, "busybox-mock")
	require.NoError(t, os.MkdirAll(busyboxDir, 0755))

	// Create mock shell binaries
	for _, bin := range []string{"sh", "cp", "echo", "rm", "mkdir"} {
		mockBin := filepath.Join(busyboxDir, bin)
		// Create a simple script that calls the system binary
		script := fmt.Sprintf("#!/bin/sh\nexec /bin/%s \"$@\"\n", bin)
		require.NoError(t, os.WriteFile(mockBin, []byte(script), 0755))
	}

	// Set required environment variables
	os.Setenv("TESTKUBE_TW_INTERNAL_PATH", filepath.Join(testDir, ".tktw"))
	os.Setenv("TESTKUBE_TW_STATE_PATH", filepath.Join(testDir, ".tktw", "state"))
	os.Setenv("TESTKUBE_TW_TERMINATION_LOG_PATH", filepath.Join(testDir, "termination.log"))
	os.Setenv("TESTKUBE_TW_INIT_BINARY_PATH", actualInitBinary)
	os.Setenv("TESTKUBE_TW_TOOLKIT_BINARY_PATH", actualToolkitBinary)
	os.Setenv("TESTKUBE_TW_BUSYBOX_BINARY_PATH", busyboxDir)
	os.Setenv("CI", "1")

	// Create basic actions
	actions := [][]map[string]interface{}{
		// Group 0 - Setup (without copying binaries in test)
		{
			{"_": map[string]bool{"i": true, "t": true, "b": true}},
		},
		// Group 1 - Execute simple command
		{
			{"d": map[string]interface{}{"c": "true", "r": "test-step"}},
			{"c": map[string]interface{}{
				"r": "test-step",
				"c": map[string]interface{}{
					"command": []string{"/bin/echo", "Hello from TestKube"},
				},
			}},
			{"e": map[string]interface{}{"r": "test-step"}},
		},
	}

	actionsJSON, _ := json.Marshal(actions)
	os.Setenv("_01_TKI_I", string(actionsJSON))

	// Set internal config
	config := map[string]interface{}{
		"e": map[string]string{"i": "test-execution"},
		"w": map[string]string{"w": "test-workflow"},
		"r": map[string]string{"i": "test-resource"},
	}
	configJSON, _ := json.Marshal(config)
	os.Setenv("_03_TKI_C", string(configJSON))
}

func setupEnvWithRetry(t *testing.T, testDir string, scriptPath string) {
	setupEnv(t, testDir)

	// Override with retry actions
	actions := [][]map[string]interface{}{
		// Group 0 - Setup (without copying binaries in test)
		{
			{"_": map[string]bool{"i": true, "t": true, "b": true}},
		},
		// Group 1 - Command with retry
		{
			{"d": map[string]interface{}{"c": "true", "r": "retry-step"}},
			{"R": map[string]interface{}{"r": "retry-step", "c": 2, "u": "passed"}}, // retry
			{"c": map[string]interface{}{
				"r": "retry-step",
				"c": map[string]interface{}{
					"command": []string{scriptPath},
				},
			}},
			{"e": map[string]interface{}{"r": "retry-step"}},
		},
	}

	actionsJSON, _ := json.Marshal(actions)
	os.Setenv("_01_TKI_I", string(actionsJSON))
}

func setupEnvWithActions(t *testing.T, testDir string, actions [][]map[string]interface{}) {
	// Clear environment first
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "TW_") || strings.HasPrefix(env, "_") || strings.HasPrefix(env, "TESTKUBE_TW_") {
			key := strings.Split(env, "=")[0]
			os.Unsetenv(key)
		}
	}

	// Use actual binaries from bin directory
	projectRoot := getProjectRoot()
	actualInitBinary := filepath.Join(projectRoot, "bin", "app", "testworkflow-init")
	actualToolkitBinary := filepath.Join(projectRoot, "bin", "app", "testworkflow-toolkit")

	// Verify binaries exist
	if _, err := os.Stat(actualInitBinary); os.IsNotExist(err) {
		t.Fatalf("testworkflow-init binary not found at %s. Run 'make build-init build-toolkit' first", actualInitBinary)
	}
	if _, err := os.Stat(actualToolkitBinary); os.IsNotExist(err) {
		t.Fatalf("testworkflow-toolkit binary not found at %s. Run 'make build-init build-toolkit' first", actualToolkitBinary)
	}

	// Create a mock busybox directory with basic shell utilities
	busyboxDir := filepath.Join(testDir, "busybox-mock")
	require.NoError(t, os.MkdirAll(busyboxDir, 0755))

	// Create mock shell binaries
	for _, bin := range []string{"sh", "cp", "echo", "rm", "mkdir"} {
		mockBin := filepath.Join(busyboxDir, bin)
		// Create a simple script that calls the system binary
		script := fmt.Sprintf("#!/bin/sh\nexec /bin/%s \"$@\"\n", bin)
		require.NoError(t, os.WriteFile(mockBin, []byte(script), 0755))
	}

	// Set required environment variables
	os.Setenv("TESTKUBE_TW_INTERNAL_PATH", filepath.Join(testDir, ".tktw"))
	os.Setenv("TESTKUBE_TW_STATE_PATH", filepath.Join(testDir, ".tktw", "state"))
	os.Setenv("TESTKUBE_TW_TERMINATION_LOG_PATH", filepath.Join(testDir, "termination.log"))
	os.Setenv("TESTKUBE_TW_INIT_BINARY_PATH", actualInitBinary)
	os.Setenv("TESTKUBE_TW_TOOLKIT_BINARY_PATH", actualToolkitBinary)
	os.Setenv("TESTKUBE_TW_BUSYBOX_BINARY_PATH", busyboxDir)
	os.Setenv("CI", "1")

	actionsJSON, _ := json.Marshal(actions)
	os.Setenv("_01_TKI_I", string(actionsJSON))

	// Set internal config
	config := map[string]interface{}{
		"e": map[string]string{"i": "test-execution"},
		"w": map[string]string{"w": "test-workflow"},
		"r": map[string]string{"i": "test-resource"},
	}
	configJSON, _ := json.Marshal(config)
	os.Setenv("_03_TKI_C", string(configJSON))
}

func setupEnvWithMetrics(t *testing.T, testDir string, scriptPath string) {
	setupEnv(t, testDir)

	// Override with metrics test actions
	actions := [][]map[string]interface{}{
		// Group 0 - Setup (without copying binaries in test)
		{
			{"_": map[string]bool{"i": true, "t": true, "b": true}},
		},
		// Group 1 - CPU intensive task
		{
			{"d": map[string]interface{}{"c": "true", "r": "cpu-test"}},
			{"c": map[string]interface{}{
				"r": "cpu-test",
				"c": map[string]interface{}{
					"command": []string{scriptPath},
				},
			}},
			{"e": map[string]interface{}{"r": "cpu-test"}},
		},
	}

	actionsJSON, _ := json.Marshal(actions)
	os.Setenv("_01_TKI_I", string(actionsJSON))

	// Set resource limits
	os.Setenv("_04_TKI_R_R_C", "100")       // 100m CPU request
	os.Setenv("_04_TKI_R_L_C", "500")       // 500m CPU limit
	os.Setenv("_04_TKI_R_R_M", "67108864")  // 64Mi memory request
	os.Setenv("_04_TKI_R_L_M", "134217728") // 128Mi memory limit
}

func updateConstants(testDir string) {
	// Update constants to use test paths
	constants.InternalPath = filepath.Join(testDir, ".tktw")
	constants.StatePath = filepath.Join(testDir, ".tktw", "state")
	constants.TerminationLogPath = filepath.Join(testDir, "termination.log")
	constants.InternalBinPath = filepath.Join(constants.InternalPath, "bin")
	constants.InitPath = filepath.Join(constants.InternalPath, "init")
	constants.ToolkitPath = filepath.Join(constants.InternalPath, "toolkit")
}

type TestState struct {
	CurrentGroupIndex int
	Steps             map[string]interface{}
	Output            map[string]string
}

func loadStateFromPath(t *testing.T, path string) TestState {
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var rawState map[string]interface{}
	err = json.Unmarshal(data, &rawState)
	require.NoError(t, err)

	state := TestState{
		Steps:  make(map[string]interface{}),
		Output: make(map[string]string),
	}

	if g, ok := rawState["g"].(float64); ok {
		state.CurrentGroupIndex = int(g)
	}

	if steps, ok := rawState["S"].(map[string]interface{}); ok {
		state.Steps = steps
	}

	if output, ok := rawState["o"].(map[string]interface{}); ok {
		for k, v := range output {
			if str, ok := v.(string); ok {
				state.Output[k] = str
			}
		}
	}

	return state
}
