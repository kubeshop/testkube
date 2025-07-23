package runner

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"github.com/kubeshop/testkube/cmd/testworkflow-init/constants"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/data"
	"github.com/kubeshop/testkube/cmd/testworkflow-init/orchestration"
)

type StateManager interface {
	EnsureStateFile() error
	LoadInitialState() error
}

type FileSystem interface {
	Stat(name string) (os.FileInfo, error)
	MkdirAll(path string, perm os.FileMode) error
	WriteFile(name string, data []byte, perm os.FileMode) error
	Chmod(name string, mode os.FileMode) error
}

type osFileSystem struct{}

func (osFileSystem) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

func (osFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (osFileSystem) WriteFile(name string, data []byte, perm os.FileMode) error {
	return os.WriteFile(name, data, perm)
}

func (osFileSystem) Chmod(name string, mode os.FileMode) error {
	return os.Chmod(name, mode)
}

type stateManager struct {
	stdout interface {
		Hint(string, string)
		SetSensitiveWords([]string)
	}
	stdoutUnsafe interface {
		Print(string)
		Error(string)
	}
	fs FileSystem
}

func NewStateManager(stdout, stdoutUnsafe interface{}) StateManager {
	return &stateManager{
		stdout: stdout.(interface {
			Hint(string, string)
			SetSensitiveWords([]string)
		}),
		stdoutUnsafe: stdoutUnsafe.(interface {
			Print(string)
			Error(string)
		}),
		fs: osFileSystem{},
	}
}

func (sm *stateManager) EnsureStateFile() error {
	_, err := sm.fs.Stat(constants.StatePath)
	if errors.Is(err, os.ErrNotExist) {
		sm.stdout.Hint(constants.InitStepName, constants.InstructionStart)
		sm.stdoutUnsafe.Print("Creating state...")

		dir := filepath.Dir(constants.StatePath)
		if err := sm.fs.MkdirAll(dir, 0777); err != nil {
			sm.stdoutUnsafe.Error(" error\n")
			return errors.Wrapf(err, "failed to create directory %s", dir)
		}

		// IMPORTANT: 0777 permissions are REQUIRED for Kubernetes shared volumes.
		// Containers in the same pod may run with different UIDs/GIDs, and the
		// state file must be readable/writable by all containers. This is NOT
		// a security issue because:
		// 1. The file is on a pod-local volume, not host filesystem
		// 2. Only containers within the same pod can access it
		// 3. All containers in the pod are part of the same TestWorkflow
		// DO NOT "FIX" THIS TO MORE RESTRICTIVE PERMISSIONS!
		if err := sm.fs.WriteFile(constants.StatePath, nil, 0777); err != nil {
			sm.stdoutUnsafe.Error(" error\n")
			return errors.Wrap(err, "failed to create state file")
		}

		sm.fs.Chmod(constants.StatePath, 0777)
		sm.stdoutUnsafe.Print(" done\n")
		return nil
	} else if err != nil {
		sm.stdout.Hint(constants.InitStepName, constants.InstructionStart)
		sm.stdoutUnsafe.Print("Accessing state...")
		sm.stdoutUnsafe.Error(" error\n")
		return errors.Wrap(err, "cannot access state file")
	}

	return nil
}

func (sm *stateManager) LoadInitialState() error {
	orchestration.Setup.UseBaseEnv()
	internalConfig := orchestration.Setup.GetInternalConfig()
	containerName := orchestration.Setup.GetContainerName()

	if len(internalConfig.Execution.SecretMountPaths) != 0 {
		secretVolumeData := orchestration.Setup.GetSecretVolumeData(internalConfig.Execution.SecretMountPaths[containerName])
		orchestration.Setup.AddSensitiveWords(secretVolumeData...)
	}

	sm.stdout.SetSensitiveWords(orchestration.Setup.GetSensitiveWords())
	actionGroups := orchestration.Setup.GetActionGroups()
	signature := orchestration.Setup.GetSignature()
	containerResources := orchestration.Setup.GetContainerResources()

	if actionGroups != nil {
		sm.stdoutUnsafe.Print("Initializing state...")
		state := data.GetState()
		state.Actions = actionGroups
		state.InternalConfig = internalConfig
		state.Signature = signature
		state.ContainerResources = containerResources
		sm.stdoutUnsafe.Print(" done\n")
	}

	return nil
}
