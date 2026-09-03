package testworkflowprocessor

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/executioncache"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

// cacheStateEnvName points both cache stages at the file they hand over through.
const cacheStateEnvName = "TK_CACHE_STATE"

// validateCache rejects a cache block that cannot work, at bundle time.
//
// These are the only cache failures that are fatal, and deliberately so: they are
// authoring mistakes with no sensible runtime fallback, and failing here surfaces them
// before a pod is ever created. Everything that can go wrong at runtime - a miss, a
// corrupt archive, unreachable storage - degrades to an uncached run instead.
func validateCache(cache *testworkflowsv1.StepCache) error {
	if cache.Key == "" {
		return errors.New("cache: a key is required")
	}
	if len(cache.Paths) == 0 {
		return errors.New("cache: there needs to be at least one path to cache")
	}
	for i, declared := range cache.Paths {
		if declared == "" {
			return fmt.Errorf("cache.paths[%d]: path is empty", i)
		}

		// A path still holding an expression cannot be honoured, and failing quietly
		// would be the worst outcome. mountCachePaths has to decide here, before the pod
		// exists, which paths need a volume of their own; the toolkit resolves the same
		// paths later, inside the pod. If the two disagree, a volume is mounted at the
		// literal template while the restore writes to whatever it resolved to - so the
		// cache would appear to work and quietly do nothing, which is the exact failure
		// the mandatory mounting exists to prevent.
		//
		// Everything resolvable before scheduling - config, workflow and execution
		// values - has already been substituted by the time this runs, so this only
		// rejects paths that depend on the pod itself.
		if !expressions.IsTemplateStringWithoutExpressions(declared) {
			return fmt.Errorf("cache.paths[%d]: %q: a cache path cannot contain an expression that is only resolvable inside the pod, because the volume to mount has to be decided before it starts; use a config value, or write the path out", i, declared)
		}
	}
	return nil
}

// isAbsContainerPath reports whether p is absolute inside the container.
//
// Not filepath.IsAbs: that answers for the host the processor happens to run on, and on
// Windows it rejects "/root/.m2". Container paths are always POSIX.
func isAbsContainerPath(p string) bool {
	return strings.HasPrefix(p, "/")
}

// mountCachePaths gives every cached path a volume shared across the step's containers.
//
// This is not an optimization, it is what makes the feature work at all. Each stage of a
// step becomes its own container, and containers in a pod share volumes but not their
// root filesystems. A path restored outside any volume would land in the restore
// container's own filesystem, where the container that runs the install cannot see it -
// so the cache would appear to work and silently do nothing.
//
// The mount is appended to the parent container so that every sibling stage of the step
// sees it, including the save stage that a later operation creates.
func mountCachePaths(layer Intermediate, container stage.Container, selfContainer stage.Container, cache *testworkflowsv1.StepCache) error {
	for i, declared := range cache.Paths {
		explicit := cache.Mount != nil
		wanted := !explicit || *cache.Mount

		// Resolve against the working directory when it is already known, so that the
		// volume check and the mount below deal in one unambiguous path.
		//
		// Resolved with `path`, not `filepath`: these are paths inside a Linux
		// container, and on a Windows host filepath.IsAbs("/root/.m2") is false, which
		// would misclassify a perfectly good absolute path as relative.
		//
		// A working directory that is not known yet is not an error. It is the common
		// case - the default container config sets none, so an image's own WORKDIR
		// decides - and a relative path left as declared is resolved against the
		// container's working directory when the volume mounts are finalized, exactly as
		// it is for the artifacts and tarball steps.
		cachePath := declared
		if !isAbsContainerPath(cachePath) {
			if wd, err := expressions.EvalTemplate(selfContainer.WorkingDir(), expressions.StdLibMachine); err == nil && isAbsContainerPath(wd) {
				cachePath = path.Join(wd, declared)
			}
		}
		cachePath = path.Clean(cachePath)

		if wanted && selfContainer.HasVolumeAt(cachePath) {
			// Already inside a volume - the repository clone, or the default /data -
			// so reusing it keeps the cache alongside the content it belongs to.
			continue
		}
		if !wanted {
			if !selfContainer.HasVolumeAt(cachePath) {
				return fmt.Errorf("cache.paths[%d]: %s: is not part of any volume: should be mounted", i, cachePath)
			}
			continue
		}

		volumeMount := layer.AddEmptyDirVolume(nil, cachePath)
		container.AppendVolumeMounts(volumeMount)
	}
	return nil
}

// cacheToolkitCommand builds the toolkit invocation, passing the step's volume mounts so
// the walker knows which roots it may read and write.
func cacheToolkitCommand(container stage.Container, verb string) []string {
	cmd := []string{constants.DefaultToolkitPath, "cache", verb}
	for _, mount := range container.VolumeMounts() {
		if mount.MountPath == constants.DefaultInternalPath {
			continue
		}
		cmd = append(cmd, "-m", strings.TrimRight(mount.MountPath, `/\`))
	}
	return cmd
}

// ProcessCacheRestore restores the cache before the step's own operations run.
//
// Registered after the content operations so that the repository is already checked out
// when the toolkit resolves the key: a key derived from a lockfile cannot be computed
// before the file exists.
func ProcessCacheRestore(_ InternalProcessor, layer Intermediate, container stage.Container, step testworkflowsv1.Step) (stage.Stage, error) {
	if step.Cache == nil {
		return nil, nil
	}
	if err := validateCache(step.Cache); err != nil {
		return nil, err
	}

	selfContainer := container.CreateChild().
		ApplyCR(&testworkflowsv1.ContainerConfig{WorkingDir: step.Cache.WorkingDir})
	self := stage.NewContainerStage(layer.NextRef(), selfContainer)
	self.SetCategory("Restore cache")

	// Pure: it only writes into mounted volumes, so it may share a container with a
	// neighbouring stage. mountCachePaths is what makes that claim true.
	self.SetPure(true)

	if err := mountCachePaths(layer, container, selfContainer, step.Cache); err != nil {
		return nil, err
	}

	// The two stages are separate containers, so the save stage cannot simply
	// recompute the key: an install may rewrite the very lockfile the key hashes -
	// npm ci does - and the entry would then be stored under a key nothing ever looks
	// up. The restore stage records what it resolved, and the save stage reads it back.
	// Setting the variable on the parent container shares it with every sibling.
	stateFile := filepath.ToSlash(filepath.Join(constants.DefaultTestkubePath, ".cache", self.Ref()+".json"))
	container.AppendEnv(corev1.EnvVar{Name: cacheStateEnvName, Value: stateFile})
	encoded, err := expressions.EncodeBase64JSON(executioncache.Args{
		Key:         step.Cache.Key,
		RestoreKeys: step.Cache.RestoreKeys,
		Paths:       step.Cache.Paths,
		Scope:       string(executioncache.ParseScope(string(step.Cache.Scope))),
		State:       stateFile,
	})
	if err != nil {
		return nil, fmt.Errorf("cache: encoding the restore arguments: %w", err)
	}

	selfContainer.
		SetImage(constants.DefaultToolkitImage).
		SetImagePullPolicy(corev1.PullIfNotPresent).
		SetCommand(cacheToolkitCommand(container, "restore")...).
		SetArgs("--base64", encoded).
		EnableToolkit(self.Ref())

	return self, nil
}

// ProcessCacheSave stores the cache once the step's operations have passed.
//
// Note that no condition is set. An empty condition inherits the step group's, which is
// "passed", and that is exactly right: a failed install must not be published under a
// content-hash key, because every later run would restore the broken tree and no user
// could invalidate it. This is the one place the behaviour deliberately differs from the
// artifacts stage, which runs on "always".
func ProcessCacheSave(_ InternalProcessor, layer Intermediate, container stage.Container, step testworkflowsv1.Step) (stage.Stage, error) {
	if step.Cache == nil {
		return nil, nil
	}
	if err := validateCache(step.Cache); err != nil {
		return nil, err
	}

	selfContainer := container.CreateChild().
		ApplyCR(&testworkflowsv1.ContainerConfig{WorkingDir: step.Cache.WorkingDir})
	self := stage.NewContainerStage(layer.NextRef(), selfContainer)
	self.SetCategory("Save cache")
	self.SetPure(true)

	// The volumes already exist: the restore stage added them to the shared parent, and
	// mounting them again here would allocate a second, empty emptyDir over the top.

	encoded, err := expressions.EncodeBase64JSON(executioncache.Args{
		Key:         step.Cache.Key,
		RestoreKeys: step.Cache.RestoreKeys,
		Paths:       step.Cache.Paths,
		Scope:       string(executioncache.ParseScope(string(step.Cache.Scope))),
	})
	if err != nil {
		return nil, fmt.Errorf("cache: encoding the save arguments: %w", err)
	}

	selfContainer.
		SetImage(constants.DefaultToolkitImage).
		SetImagePullPolicy(corev1.PullIfNotPresent).
		SetCommand(cacheToolkitCommand(container, "save")...).
		SetArgs("--base64", encoded).
		EnableToolkit(self.Ref())

	return self, nil
}
