package presets

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/executioncache"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
)

// cacheStateEnvName is the variable the two cache stages hand the resolved key over in.
const cacheStateEnvName = "TK_CACHE_STATE"

func bundleWithCache(t *testing.T, step testworkflowsv1.Step) (*testworkflowprocessor.Bundle, error) {
	t.Helper()
	wf := &testworkflowsv1.TestWorkflow{
		Spec: testworkflowsv1.TestWorkflowSpec{Steps: []testworkflowsv1.Step{step}},
	}
	return proc.Bundle(context.Background(), wf, testworkflowprocessor.BundleOptions{Config: testConfig})
}

// stageCommand is one stage's rendered command line.
//
// The cache stages are pure, so they are normally merged into a neighbouring container
// and their commands live in the actions spec rather than in a container of their own -
// which is why these assertions read the actions, not the pod spec.
type stageCommand struct {
	Ref  string
	Line string
	Env  []corev1.EnvVar
}

func stageCommands(res *testworkflowprocessor.Bundle) []stageCommand {
	var stages []stageCommand
	for _, group := range res.Actions() {
		for _, action := range group {
			if action.Container == nil {
				continue
			}
			var parts []string
			if action.Container.Config.Command != nil {
				parts = append(parts, *action.Container.Config.Command...)
			}
			if action.Container.Config.Args != nil {
				parts = append(parts, *action.Container.Config.Args...)
			}
			env := make([]corev1.EnvVar, 0, len(action.Container.Config.Env))
			for _, envVar := range action.Container.Config.Env {
				env = append(env, envVar.EnvVar)
			}
			stages = append(stages, stageCommand{
				Ref:  action.Container.Ref,
				Line: strings.Join(parts, " "),
				Env:  env,
			})
		}
	}
	return stages
}

// cachePayload decodes the base64 argument a cache stage carries.
func cachePayload(t *testing.T, line string) executioncache.Args {
	t.Helper()
	_, encoded, found := strings.Cut(line, "--base64 ")
	require.True(t, found, "the stage should carry a base64 payload: %s", line)
	encoded = strings.Fields(encoded)[0]

	var args executioncache.Args
	require.NoError(t, expressions.DecodeBase64JSON(encoded, &args))
	return args
}

// TestProcessCache_StageOrder pins where the two stages sit. Restore has to follow the
// git clone, or the key cannot hash a lockfile and the clone would overwrite whatever
// was restored; save has to follow the step's own work, or there is nothing to store.
func TestProcessCache_StageOrder(t *testing.T) {
	res, err := bundleWithCache(t, testworkflowsv1.Step{
		StepSource: testworkflowsv1.StepSource{
			Content: &testworkflowsv1.Content{
				Git: &testworkflowsv1.ContentGit{Uri: "https://example.com/repo.git"},
			},
		},
		StepOperations: testworkflowsv1.StepOperations{
			Shell: "npm ci",
			Cache: &testworkflowsv1.StepCache{
				Key:   `npm-{{ hash_files("package-lock.json") }}`,
				Paths: []string{"/data/repo/node_modules"},
			},
			Artifacts: &testworkflowsv1.StepArtifacts{Paths: []string{"*.xml"}},
		},
	})
	require.NoError(t, err)

	var order []string
	for _, stage := range stageCommands(res) {
		switch {
		case strings.Contains(stage.Line, "cache restore"):
			order = append(order, "restore")
		case strings.Contains(stage.Line, "cache save"):
			order = append(order, "save")
		case strings.Contains(stage.Line, "clone"):
			order = append(order, "clone")
		case strings.Contains(stage.Line, "artifacts"):
			order = append(order, "artifacts")
		}
	}

	assert.Equal(t, []string{"clone", "restore", "save", "artifacts"}, order)
}

// TestProcessCache_KeyTemplateStaysOpaque guards the whole non-fatality story.
// testworkflow-init resolves every container argument with FinalizerFail and exits the
// step on failure, so if the key template ever reached the arguments in the clear, a
// missing lockfile would kill the step instead of simply missing the cache.
func TestProcessCache_KeyTemplateStaysOpaque(t *testing.T) {
	res, err := bundleWithCache(t, testworkflowsv1.Step{
		StepOperations: testworkflowsv1.StepOperations{
			Shell: "npm ci",
			Cache: &testworkflowsv1.StepCache{
				Key:   `npm-{{ hash_files("package-lock.json") }}`,
				Paths: []string{"node_modules"},
			},
		},
	})
	require.NoError(t, err)

	var found int
	for _, stage := range stageCommands(res) {
		if !strings.Contains(stage.Line, "cache restore") && !strings.Contains(stage.Line, "cache save") {
			continue
		}
		found++
		assert.NotContains(t, stage.Line, "hash_files", "the key template must not be readable in the arguments")
		assert.NotContains(t, stage.Line, "{{", "the key template must not be readable in the arguments")

		// It does survive inside the payload, unresolved, for the toolkit to handle.
		// Simplify normalises the whitespace inside the braces, so match on the shape.
		payloadKey := cachePayload(t, stage.Line).Key
		assert.Contains(t, payloadKey, "hash_files")
		assert.Contains(t, payloadKey, "{{")
		assert.True(t, strings.HasPrefix(payloadKey, "npm-"))
	}
	assert.Equal(t, 2, found, "both cache stages should be present")
}

// TestProcessCache_SharesTheStateFile covers the handshake. The two stages run in
// separate containers and an install may rewrite the lockfile the key hashes, so the
// save stage has to read back the key the restore stage actually resolved.
func TestProcessCache_SharesTheStateFile(t *testing.T) {
	res, err := bundleWithCache(t, testworkflowsv1.Step{
		StepOperations: testworkflowsv1.StepOperations{
			Shell: "npm ci",
			Cache: &testworkflowsv1.StepCache{Key: "npm-abc", Paths: []string{"node_modules"}},
		},
	})
	require.NoError(t, err)

	// The restore stage names the file in its payload...
	var stateFromPayload string
	for _, stage := range stageCommands(res) {
		if strings.Contains(stage.Line, "cache restore") {
			stateFromPayload = cachePayload(t, stage.Line).State
		}
	}
	require.NotEmpty(t, stateFromPayload, "the restore stage must know where to record its key")
	assert.Contains(t, stateFromPayload, ".cache",
		"the state file should live in its own directory")

	// ...and the save stage finds it through the shared environment, since it is given
	// no state path of its own. The variable is emitted once per stage group under a
	// scoped name (_0_, _1_, ...), so match on the suffix rather than the bare name.
	var envValues []string
	for _, container := range append(res.Job.Spec.Template.Spec.InitContainers, res.Job.Spec.Template.Spec.Containers...) {
		for _, envVar := range container.Env {
			if strings.HasSuffix(envVar.Name, cacheStateEnvName) {
				envValues = append(envValues, envVar.Value)
			}
		}
	}

	require.NotEmpty(t, envValues, "the stages need a shared location to hand the key over")
	// Every stage of the step, not just the two cache ones: appending to the shared
	// parent container is what guarantees the save stage sees the restore stage's file.
	assert.GreaterOrEqual(t, len(envValues), 2)
	for _, value := range envValues {
		assert.Equal(t, stateFromPayload, value, "every stage must agree on the state file")
	}
}

// TestProcessCache_MountsUncoveredPaths covers the failure mode that would make the
// feature silently useless: each stage is its own container, and containers share
// volumes but not their root filesystems, so a path outside every volume is restored
// where the container running the install cannot see it.
func TestProcessCache_MountsUncoveredPaths(t *testing.T) {
	// Counts the volumes the cached paths got, which is what these assertions are about.
	// The save stage's staging volume is left out deliberately: it is one per cached
	// step regardless of how many paths there are, so counting it would add a constant
	// to every expectation below and stop the numbers describing anything.
	emptyDirCount := func(res *testworkflowprocessor.Bundle) int {
		staging := map[string]struct{}{}
		for _, container := range res.Job.Spec.Template.Spec.Containers {
			for _, mount := range container.VolumeMounts {
				if mount.MountPath == cacheTempDirPath {
					staging[mount.Name] = struct{}{}
				}
			}
		}

		var count int
		for _, volume := range res.Job.Spec.Template.Spec.Volumes {
			if _, isStaging := staging[volume.Name]; isStaging {
				continue
			}
			if volume.EmptyDir != nil {
				count++
			}
		}
		return count
	}

	build := func(t *testing.T, cache *testworkflowsv1.StepCache) *testworkflowprocessor.Bundle {
		t.Helper()
		res, err := bundleWithCache(t, testworkflowsv1.Step{
			StepOperations: testworkflowsv1.StepOperations{Shell: "mvn verify", Cache: cache},
		})
		require.NoError(t, err)
		return res
	}

	// The baseline is the same step with no cache at all, so the assertions describe
	// what the cache adds rather than hard-coding how many volumes a pod happens to get.
	baselineRes, err := bundleWithCache(t, testworkflowsv1.Step{
		StepOperations: testworkflowsv1.StepOperations{Shell: "mvn verify"},
	})
	require.NoError(t, err)
	baseline := emptyDirCount(baselineRes)

	t.Run("a path outside any volume gets one", func(t *testing.T) {
		res := build(t, &testworkflowsv1.StepCache{Key: "m2-abc", Paths: []string{"/root/.m2"}})
		assert.Equal(t, baseline+1, emptyDirCount(res),
			"an uncovered path needs a volume, or the install container cannot see the restore")
	})

	t.Run("a path already inside a shared volume reuses it", func(t *testing.T) {
		res := build(t, &testworkflowsv1.StepCache{Key: "npm-abc", Paths: []string{"/data/node_modules"}})
		assert.Equal(t, baseline, emptyDirCount(res),
			"/data is already shared, so no second volume should be added")
	})

	t.Run("one volume per path, not one per stage", func(t *testing.T) {
		// Adding it in both stages would lay a second, empty emptyDir over the first
		// and lose everything the restore had written.
		res := build(t, &testworkflowsv1.StepCache{Key: "m2-abc", Paths: []string{"/root/.m2", "/root/.npm"}})
		assert.Equal(t, baseline+2, emptyDirCount(res))
	})
}

func TestProcessCache_Rejects(t *testing.T) {
	build := func(cache *testworkflowsv1.StepCache) error {
		_, err := bundleWithCache(t, testworkflowsv1.Step{
			StepOperations: testworkflowsv1.StepOperations{Shell: "true", Cache: cache},
		})
		return err
	}

	// Authoring mistakes with no sensible runtime fallback, so they fail before a pod
	// exists - unlike every runtime cache problem, which degrades to a miss.
	assert.ErrorContains(t, build(&testworkflowsv1.StepCache{Paths: []string{"node_modules"}}), "key is required")
	assert.ErrorContains(t, build(&testworkflowsv1.StepCache{Key: "k"}), "at least one path")
	assert.ErrorContains(t, build(&testworkflowsv1.StepCache{Key: "k", Paths: []string{""}}), "path is empty")

	// mount: false on a path in no volume would restore into a container's own
	// filesystem and silently do nothing, so it is refused rather than accepted.
	assert.ErrorContains(t, build(&testworkflowsv1.StepCache{
		Key: "k", Paths: []string{"/root/.m2"}, Mount: common.Ptr(false),
	}), "should be mounted")
}

// TestProcessCache_WrapsNestedSteps covers a cache declared on a step that has children
// rather than a command of its own.
//
// The shape works, and it works for a reason rather than by luck: ProcessCacheRestore is
// registered before every operation that produces execution - ProcessNestedSteps
// included - and ProcessCacheSave after all of them, so the children land between a
// restore and a save that were already positioned to wrap whatever executes. None of
// that is stated anywhere it would be checked, though. An operation inserted between the
// restore and ProcessNestedSteps, or a reordering of that list, would silently move the
// children outside the cache, and every other test here uses a leaf step.
//
// This is also the most useful shape in practice: one cache around an install and the
// build that consumes it, rather than one per step.
func TestProcessCache_WrapsNestedSteps(t *testing.T) {
	res, err := bundleWithCache(t, testworkflowsv1.Step{
		StepOperations: testworkflowsv1.StepOperations{
			Cache: &testworkflowsv1.StepCache{Key: "m2-abc", Paths: []string{"/root/.m2"}},
		},
		Steps: []testworkflowsv1.Step{
			{StepOperations: testworkflowsv1.StepOperations{Shell: "mvn verify"}},
			{StepOperations: testworkflowsv1.StepOperations{Shell: "mvn test"}},
		},
	})
	require.NoError(t, err)

	var restore, save stageCommand
	var children []stageCommand
	for _, stage := range stageCommands(res) {
		switch {
		case strings.Contains(stage.Line, "cache restore"):
			restore = stage
		case strings.Contains(stage.Line, "cache save"):
			save = stage
		case strings.Contains(stage.Line, "mvn "):
			children = append(children, stage)
		}
	}
	require.NotEmpty(t, restore.Ref, "there should be a restore stage")
	require.NotEmpty(t, save.Ref, "there should be a save stage")
	require.Len(t, children, 2, "both children should be present")

	// Order: the restore has to precede every child and the save has to follow them, or
	// the children run against a tree that was not restored yet, or was packed too early.
	order := make(map[string]int)
	for i, stage := range stageCommands(res) {
		order[stage.Ref] = i
	}
	for _, child := range children {
		assert.Less(t, order[restore.Ref], order[child.Ref], "the restore must precede every child")
		assert.Less(t, order[child.Ref], order[save.Ref], "the save must follow every child")
	}

	// A failing child must not publish a cache. The save carries no condition of its own,
	// so it inherits the group's, which conjoins every stage before it - and that is what
	// makes the guarantee stated for a leaf step hold for this shape too: an entry is
	// immutable, so a tree published by a failed build could never be corrected.
	conditions := make(map[string]string)
	for _, group := range res.Actions() {
		for _, action := range group {
			if action.Declare != nil {
				conditions[action.Declare.Ref] = action.Declare.Condition
			}
		}
	}
	require.Contains(t, conditions, save.Ref)
	for _, child := range children {
		assert.Contains(t, conditions[save.Ref], child.Ref,
			"the save has to depend on every child, or a failed one still publishes an entry")
	}

	// And the children have to be able to see what was restored. The volume goes on the
	// parent container precisely so every stage of the step inherits it.
	var mounted bool
	for _, container := range res.Job.Spec.Template.Spec.Containers {
		for _, mount := range container.VolumeMounts {
			if mount.MountPath == "/root/.m2" {
				mounted = true
			}
		}
	}
	assert.True(t, mounted, "the cached path has to be mounted where the children run")
}

// TestProcessCache_OnAParallelStepTravelsToTheWorkers covers where a cache has to be
// declared when parallel workers are involved, which is not where a reader expects.
//
// A cache works by mounting volumes that the restore writes into and the save reads
// back. Workers run in pods of their own, and ProcessParallel nils out VolumeMounts when
// a worker inherits the container config - so a cache on an ancestor of a parallel step
// restores into the pod that launches the workers and saves from it, doing nothing for
// the pods that do the work. It looks like it is caching.
//
// Declared on the parallel step itself it does reach them: the whole StepParallel is
// encoded into the toolkit's argument, StepOperations included, and the toolkit moves
// those operations into each worker's first step. Both halves are asserted from that one
// payload, because both are properties of what gets encoded.
func TestProcessCache_OnAParallelStepTravelsToTheWorkers(t *testing.T) {
	res, err := bundleWithCache(t, testworkflowsv1.Step{
		StepOperations: testworkflowsv1.StepOperations{
			Shell: "echo before",
		},
		Parallel: &testworkflowsv1.StepParallel{
			StepOperations: testworkflowsv1.StepOperations{
				Shell: "npm test",
				Cache: &testworkflowsv1.StepCache{
					Key:   "npm-abc",
					Paths: []string{"/root/.npm/_cacache"},
				},
			},
		},
	})
	require.NoError(t, err)

	var encoded string
	for _, stage := range stageCommands(res) {
		if strings.Contains(stage.Line, "toolkit parallel") {
			_, rest, found := strings.Cut(stage.Line, "--base64 ")
			require.True(t, found, "the parallel stage should carry a base64 spec: %s", stage.Line)
			encoded = strings.Fields(rest)[0]
		}
	}
	require.NotEmpty(t, encoded, "there should be a parallel stage")

	var parallel testworkflowsv1.StepParallel
	require.NoError(t, expressions.DecodeBase64JSON(encoded, &parallel))

	// The cache reaches the workers, because it is part of what gets encoded.
	require.NotNil(t, parallel.Cache, "a cache on the parallel step has to travel to the workers")
	assert.Equal(t, "npm-abc", parallel.Cache.Key)
	assert.Equal(t, []string{"/root/.npm/_cacache"}, parallel.Cache.Paths)

	// And the pod's volumes do not. This is the half that makes a cache on an ancestor
	// ineffective rather than merely redundant: a worker inherits the container config
	// without its mounts, so there is nothing for a restore in the parent pod to reach.
	require.NotNil(t, parallel.Container,
		"the worker inherits a container config, so a nil one means this assertion is not being made")
	assert.Empty(t, parallel.Container.VolumeMounts,
		"a worker must not inherit the parent pod's mounts, or a cache on an ancestor would appear to reach it")
}

// TestProcessCache_StagesTheArchiveOnItsOwnVolume covers the volume the save stage
// writes its archive to before uploading it.
//
// It has to be a real mount in the pod spec, and it has to be its own volume. Sharing
// /tmp - which every container already has - would put a multi-gigabyte archive under
// the same size limit as whatever the step itself writes there, and exceeding a volume's
// size limit evicts the pod, which is the one outcome a cache must never cause.
//
// Read from the pod spec rather than the actions, because that is the distinction that
// matters here: the cache stages are pure and get merged into a neighbouring container,
// so a volume mount added to the stage's own container would not survive, while its env
// would - leaving TK_CACHE_TMP_DIR pointing at a directory that does not exist.
func TestProcessCache_StagesTheArchiveOnItsOwnVolume(t *testing.T) {
	res, err := bundleWithCache(t, testworkflowsv1.Step{
		StepOperations: testworkflowsv1.StepOperations{
			Shell: "npm ci",
			Cache: &testworkflowsv1.StepCache{Key: "k", Paths: []string{"/root/.m2"}},
		},
	})
	require.NoError(t, err)

	var mounted *corev1.VolumeMount
	for _, container := range res.Job.Spec.Template.Spec.Containers {
		for i, mount := range container.VolumeMounts {
			if mount.MountPath == cacheTempDirPath {
				mounted = &container.VolumeMounts[i]
			}
		}
	}
	require.NotNil(t, mounted, "the staging directory has to be mounted in the pod, not just named in an env var")
	assert.NotEqual(t, "/tmp", mounted.MountPath)

	// Its own volume, sized from the same limit the save refuses at, so the archive
	// cannot overrun the volume before the toolkit gets to decline it.
	var source *corev1.Volume
	for i, volume := range res.Job.Spec.Template.Spec.Volumes {
		if volume.Name == mounted.Name {
			source = &res.Job.Spec.Template.Spec.Volumes[i]
		}
	}
	require.NotNil(t, source)
	require.NotNil(t, source.EmptyDir, "staging belongs on an emptyDir, not the container's own filesystem")
	require.NotNil(t, source.EmptyDir.SizeLimit)
	assert.Equal(t, executioncache.MaxArchiveSize+cacheTempDirHeadroom, source.EmptyDir.SizeLimit.Value())

	// The save stage is pointed at it, and told the same limit the volume was sized from.
	var save stageCommand
	for _, s := range stageCommands(res) {
		if strings.Contains(s.Line, "cache save") {
			save = s
		}
	}
	require.NotEmpty(t, save.Line, "there should be a cache save stage")
	assert.Contains(t, save.Line, "--max-size "+strconv.FormatInt(executioncache.MaxArchiveSize, 10))

	// Read from the pod, and by suffix: a stage's environment is emitted into the
	// container under a group-scoped name (_0_, _1_, ...), which testworkflow-init
	// strips back to the bare name before running the stage. Same reasoning as
	// TestProcessCache_SharesTheStateFile.
	var staging []string
	for _, container := range append(res.Job.Spec.Template.Spec.InitContainers, res.Job.Spec.Template.Spec.Containers...) {
		for _, envVar := range container.Env {
			if strings.HasSuffix(envVar.Name, "TK_CACHE_TMP_DIR") {
				staging = append(staging, envVar.Value)
			}
		}
	}
	assert.Contains(t, staging, cacheTempDirPath, "the save stage has to be told where to stage")

	// And the walker is not told it may read the staging directory: an archive that is a
	// candidate for packing into itself is a trap worth closing by construction.
	assert.NotContains(t, save.Line, "-m "+cacheTempDirPath)
}

// TestProcessCache_RejectsTemplatedPaths covers a hole in the mandatory mounting.
//
// Which paths need a volume of their own has to be decided here, before the pod exists,
// while the toolkit resolves the same paths later inside it. A path still holding a
// pod-side expression would get a volume mounted at the literal template while the
// restore wrote to whatever it resolved to - so the cache would appear to work and
// quietly do nothing, which is precisely what the mounting exists to prevent.
func TestProcessCache_RejectsTemplatedPaths(t *testing.T) {
	build := func(paths []string) error {
		_, err := bundleWithCache(t, testworkflowsv1.Step{
			StepOperations: testworkflowsv1.StepOperations{
				Shell: "true",
				Cache: &testworkflowsv1.StepCache{Key: "k", Paths: paths},
			},
		})
		return err
	}

	for _, declared := range []string{
		"{{ env.HOME }}/.m2",
		"/data/{{ env.DIR }}",
		"{{ step.x.outputs.dir }}",
	} {
		assert.ErrorContains(t, build([]string{declared}), "cannot contain an expression",
			"%q depends on the pod, so the volume to mount cannot be decided", declared)
	}

	// A concrete path is fine, and so is one built from values substituted before the
	// pod is scheduled - by the time this runs those are already literals.
	assert.NoError(t, build([]string{"/root/.m2"}))
	assert.NoError(t, build([]string{"node_modules"}))

	// The key is the opposite case and has to stay templatable: resolving it needs the
	// repository, which only exists inside the pod.
	_, err := bundleWithCache(t, testworkflowsv1.Step{
		StepOperations: testworkflowsv1.StepOperations{
			Shell: "true",
			Cache: &testworkflowsv1.StepCache{
				Key:   `npm-{{ hash_files("package-lock.json") }}`,
				Paths: []string{"node_modules"},
			},
		},
	})
	assert.NoError(t, err)
}

// TestProcessCache_RejectsATemplatedWorkingDir is the same hole one level up.
//
// A relative cached path is resolved against the working directory, so the base has to
// be decidable here too. mountCachePaths only uses it when it evaluates, and would
// otherwise silently fall back to a different base than the toolkit arrives at inside
// the pod - leaving a volume mounted somewhere the restore never writes.
func TestProcessCache_RejectsATemplatedWorkingDir(t *testing.T) {
	build := func(workingDir string) error {
		_, err := bundleWithCache(t, testworkflowsv1.Step{
			StepOperations: testworkflowsv1.StepOperations{
				Shell: "true",
				Cache: &testworkflowsv1.StepCache{
					Key:        "k",
					Paths:      []string{"node_modules"},
					WorkingDir: common.Ptr(workingDir),
				},
			},
		})
		return err
	}

	for _, declared := range []string{
		"{{ env.HOME }}/app",
		"/src/{{ step.x.outputs.dir }}",
	} {
		assert.ErrorContains(t, build(declared), "cannot contain an expression",
			"%q depends on the pod, so the base for a relative path cannot be decided", declared)
	}

	// A concrete base is fine, and so is no override at all.
	assert.NoError(t, build("/src"))

	_, err := bundleWithCache(t, testworkflowsv1.Step{
		StepOperations: testworkflowsv1.StepOperations{
			Shell: "true",
			Cache: &testworkflowsv1.StepCache{Key: "k", Paths: []string{"node_modules"}},
		},
	})
	assert.NoError(t, err)
}

// TestProcessCache_ScopeDefaultsToWorkflow: an omitted or unrecognised scope must
// resolve to the narrowest one. Widening sharing is a trust decision, so it only ever
// happens because a workflow asked for it explicitly.
func TestProcessCache_ScopeDefaultsToWorkflow(t *testing.T) {
	for _, declared := range []testworkflowsv1.CacheScope{"", "nonsense"} {
		res, err := bundleWithCache(t, testworkflowsv1.Step{
			StepOperations: testworkflowsv1.StepOperations{
				Shell: "npm ci",
				Cache: &testworkflowsv1.StepCache{
					Key: "npm-abc", Paths: []string{"node_modules"}, Scope: declared,
				},
			},
		})
		require.NoError(t, err)

		for _, stage := range stageCommands(res) {
			if strings.Contains(stage.Line, "cache restore") {
				assert.Equal(t, string(executioncache.ScopeWorkflow), cachePayload(t, stage.Line).Scope,
					"scope %q should not widen sharing", declared)
			}
		}
	}

	res, err := bundleWithCache(t, testworkflowsv1.Step{
		StepOperations: testworkflowsv1.StepOperations{
			Shell: "npm ci",
			Cache: &testworkflowsv1.StepCache{
				Key: "npm-abc", Paths: []string{"node_modules"}, Scope: testworkflowsv1.CacheScopeEnvironment,
			},
		},
	})
	require.NoError(t, err)
	for _, stage := range stageCommands(res) {
		if strings.Contains(stage.Line, "cache restore") {
			assert.Equal(t, string(executioncache.ScopeEnvironment), cachePayload(t, stage.Line).Scope)
		}
	}
}

// TestProcessCache_SaveIsConditional: an empty condition inherits the step group's,
// which is "passed". Publishing a failed install under a content-hash key would poison
// every later run with no way for a user to invalidate it, which is why the cache stages
// deliberately do not copy the artifacts stage's "always".
func TestProcessCache_SaveIsConditional(t *testing.T) {
	res, err := bundleWithCache(t, testworkflowsv1.Step{
		StepOperations: testworkflowsv1.StepOperations{
			Shell: "npm ci",
			Cache: &testworkflowsv1.StepCache{Key: "npm-abc", Paths: []string{"node_modules"}},
			// Artifacts sit alongside on purpose: they carry "always", and the cache
			// stages must not have picked that up from being neighbours.
			Artifacts: &testworkflowsv1.StepArtifacts{Paths: []string{"*.xml"}},
		},
	})
	require.NoError(t, err)

	conditions := map[string]string{}
	for _, group := range res.Actions() {
		for _, action := range group {
			if action.Declare != nil {
				conditions[action.Declare.Ref] = action.Declare.Condition
			}
		}
	}

	// The signature is a tree - the step's stages hang off a root - so walk it.
	refsByCategory := map[string]string{}
	var walk func(signatures []stage.Signature)
	walk = func(signatures []stage.Signature) {
		for _, signature := range signatures {
			if category := signature.Category(); category != "" {
				refsByCategory[category] = signature.Ref()
			}
			walk(signature.Children())
		}
	}
	walk(res.FullSignature)

	saveRef, ok := refsByCategory["Save cache"]
	require.True(t, ok, "the save stage should appear in the signature")
	saveCondition, ok := conditions[saveRef]
	require.True(t, ok, "the save stage should be declared")

	// The whole point: the save stage waits on what came before it in the step. The
	// condition is expressed as the preceding stages' refs, each of which evaluates to
	// whether that stage succeeded.
	assert.NotEqual(t, "true", saveCondition,
		"a failed install must not be published under a content-hash key")
	shellRef, ok := refsByCategory["Run shell command"]
	require.True(t, ok, "the shell stage should appear in the signature")
	assert.Contains(t, saveCondition, shellRef,
		"saving must depend on the install stage having succeeded")

	// The two contrasts that give that meaning. Restore runs unconditionally, because
	// nothing precedes it that could have failed...
	restoreRef, ok := refsByCategory["Restore cache"]
	require.True(t, ok, "the restore stage should appear in the signature")
	assert.Equal(t, "true", conditions[restoreRef])
	assert.Contains(t, saveCondition, restoreRef,
		"saving must also depend on the restore stage, which precedes it")

	// ...and artifacts are deliberately unconditional even after a failure, which is
	// exactly what the cache stages must not copy.
	artifactRef, ok := refsByCategory["Upload artifacts"]
	require.True(t, ok)
	assert.Equal(t, "true", conditions[artifactRef])
}

// Mirrors the processor's own constants, so a change to either has to be deliberate.
const (
	cacheTempDirPath     = "/.tktw-cache"
	cacheTempDirHeadroom = 64 << 20
)

// TestProcessCache_RejectsTheRootHoweverItIsReached covers the gap between the two places
// a path is judged.
//
// validateCache sees only what the author declared, while mountCachePaths sees what that
// resolves to against the working directory - and a relative path reaches the root
// without ever looking like it. "." under a workingDir of "/" cleans to "." in the first
// and to "/" in the second, so the declared-path check passed it through and an empty
// volume was mounted over the container root, hiding the image's own filesystem from the
// step it was supposed to be caching for.
func TestProcessCache_RejectsTheRootHoweverItIsReached(t *testing.T) {
	build := func(workingDir string, paths ...string) error {
		cache := &testworkflowsv1.StepCache{Key: "k", Paths: paths}
		if workingDir != "" {
			cache.WorkingDir = common.Ptr(workingDir)
		}
		_, err := bundleWithCache(t, testworkflowsv1.Step{
			StepOperations: testworkflowsv1.StepOperations{Shell: "true", Cache: cache},
		})
		return err
	}

	t.Run("declared as the root", func(t *testing.T) {
		// The case that was already caught, kept so the two routes stay covered together.
		assert.ErrorContains(t, build("", "/"), "container root")
	})

	t.Run("resolved to the root through the working directory", func(t *testing.T) {
		for _, declared := range []string{".", "./", "./."} {
			assert.ErrorContains(t, build("/", declared), "container root",
				"%q under a workingDir of / resolves to the root", declared)
		}
	})

	t.Run("a path that merely sits near the root is still fine", func(t *testing.T) {
		assert.NoError(t, build("/", "root/.npm"))
		assert.NoError(t, build("/root", "."),
			"a relative path is only the root when the working directory is")
	})
}

// TestProcessCache_RejectsGlobPaths closes the gap between how the two halves of the
// feature read a cached path.
//
// mountCachePaths mounts a volume at the literal string; the toolkit hands the same
// string to the walker, which reads it as a glob and derives a search root from it. So
// "/data/c*" mounted a directory actually named "c*" while packing everything else under
// /data that happened to match - putting files nobody declared into the archive, and
// under `scope: environment` into a cache other workflows restore from.
func TestProcessCache_RejectsGlobPaths(t *testing.T) {
	build := func(paths ...string) error {
		_, err := bundleWithCache(t, testworkflowsv1.Step{
			StepOperations: testworkflowsv1.StepOperations{
				Shell: "true",
				Cache: &testworkflowsv1.StepCache{Key: "k", Paths: paths},
			},
		})
		return err
	}

	for _, declared := range []string{
		"/data/c*",
		"/data/?ache",
		"/data/[abc]ache",
		"/data/{a,b}",
		`/data/c\*`,
		"/data/**/node_modules",
	} {
		assert.ErrorContains(t, build(declared), "not a pattern",
			"%q is read as a glob when the archive is packed", declared)
	}

	// Ordinary paths, including the punctuation a real cache directory carries.
	for _, declared := range []string{
		"/root/.npm/_cacache",
		"/root/go/pkg/mod/cache/download",
		"node_modules",
		"/data/my-cache_dir.v2",
	} {
		assert.NoError(t, build(declared), "%q is a plain directory", declared)
	}
}

// TestProcessCache_SaveWaitsOnSuccessUnderAnExplicitCondition is the same guarantee for
// the shape that used to lose it.
//
// A step with no condition gets a group condition of "passed", and the save inherits that.
// A step that declares one gets a group of "true" instead, with the declared condition
// pushed down onto every child - so the save's condition became the author's alone. Under
// `condition: always` the save would then run after a failed command and publish whatever
// it left behind, under a key that can never be rewritten and that every later run
// restores.
func TestProcessCache_SaveWaitsOnSuccessUnderAnExplicitCondition(t *testing.T) {
	res, err := bundleWithCache(t, testworkflowsv1.Step{
		StepMeta: testworkflowsv1.StepMeta{Condition: "always"},
		StepOperations: testworkflowsv1.StepOperations{
			Shell: "npm ci",
			Cache: &testworkflowsv1.StepCache{Key: "npm-abc", Paths: []string{"node_modules"}},
		},
	})
	require.NoError(t, err)

	conditions := map[string]string{}
	for _, group := range res.Actions() {
		for _, action := range group {
			if action.Declare != nil {
				conditions[action.Declare.Ref] = action.Declare.Condition
			}
		}
	}

	refsByCategory := map[string]string{}
	var walk func(signatures []stage.Signature)
	walk = func(signatures []stage.Signature) {
		for _, signature := range signatures {
			if category := signature.Category(); category != "" {
				refsByCategory[category] = signature.Ref()
			}
			walk(signature.Children())
		}
	}
	walk(res.FullSignature)

	saveRef, ok := refsByCategory["Save cache"]
	require.True(t, ok, "the save stage should appear in the signature")
	saveCondition, ok := conditions[saveRef]
	require.True(t, ok, "the save stage should be declared")

	assert.NotEqual(t, "always", saveCondition,
		"the author's condition must not be the only thing gating the save")
	shellRef, ok := refsByCategory["Run shell command"]
	require.True(t, ok, "the shell stage should appear in the signature")
	assert.Contains(t, saveCondition, shellRef,
		"saving has to depend on the command having succeeded, whatever the step's own condition says")

	// The declared condition is compiled into stage references rather than kept as the
	// word, so there is nothing literal to look for - what matters is that the save is
	// gated on a stage result at all, rather than on the author.s condition alone.
}
