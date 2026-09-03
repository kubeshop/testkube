package presets

import (
	"context"
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
	emptyDirCount := func(res *testworkflowprocessor.Bundle) int {
		var count int
		for _, volume := range res.Job.Spec.Template.Spec.Volumes {
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
