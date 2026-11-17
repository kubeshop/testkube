package controlplaneresources

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	commonv1 "github.com/kubeshop/testkube/api/common/v1"
	insightsv1 "github.com/kubeshop/testkube/api/insights/v1alpha1"
	"github.com/kubeshop/testkube/pkg/controlplane/declarative"
	"github.com/kubeshop/testkube/pkg/controlplane/declarative/client"
)

type fakeClient struct {
	applied  []declarative.Object
	deleted  []string
	stored   map[string]declarative.Object
	applyErr map[string]error
	err      error
}

func (f *fakeClient) Get(_ context.Context, _ declarative.Kind, name string) (declarative.Object, error) {
	if obj, ok := f.stored[name]; ok {
		return obj, nil
	}
	return nil, client.ErrNotFound
}

func (f *fakeClient) List(context.Context, declarative.Kind) ([]declarative.Object, error) {
	var out []declarative.Object
	for _, name := range []string{"a", "b"} {
		if obj, ok := f.stored[name]; ok {
			out = append(out, obj)
		}
	}
	return out, f.err
}

func (f *fakeClient) Apply(_ context.Context, obj declarative.Object) (declarative.Object, error) {
	if err := f.applyErr[obj.GetName()]; err != nil {
		return nil, err
	}
	f.applied = append(f.applied, obj)
	return obj, nil
}

func (f *fakeClient) Delete(_ context.Context, _ declarative.Kind, name string) error {
	if _, ok := f.stored[name]; !ok {
		return client.ErrNotFound
	}
	f.deleted = append(f.deleted, name)
	return nil
}

const twoBoards = `apiVersion: insights.testkube.io/v1alpha1
kind: Board
metadata:
  name: first
spec:
  reports:
    - id: r
      kind: pass-fail
---
apiVersion: insights.testkube.io/v1alpha1
kind: Board
metadata:
  name: second
spec: {}
`

func read(t *testing.T, input string) []declarative.Object {
	t.Helper()
	objects, err := ReadInputs([]string{"-"}, strings.NewReader(input))
	require.NoError(t, err)
	return objects
}

func TestApply_MultiDocument(t *testing.T) {
	c := &fakeClient{}
	var out bytes.Buffer
	require.NoError(t, Apply(context.Background(), c, read(t, twoBoards), ApplyOptions{}, &out))

	require.Len(t, c.applied, 2)
	assert.Equal(t, "board/first applied\nboard/second applied\n", out.String())
	// What is sent is normalized.
	first := c.applied[0].(*insightsv1.Board)
	assert.Equal(t, "first", first.Spec.DisplayName)
	assert.JSONEq(t, `{"duration":"month","measure":"ratio","filter":[]}`, string(first.Spec.Reports[0].Params.Raw))
}

func TestApply_DryRunSendsNothing(t *testing.T) {
	var out bytes.Buffer
	// A nil client proves nothing is sent: any call would panic.
	require.NoError(t, Apply(context.Background(), nil, read(t, twoBoards), ApplyOptions{DryRun: true}, &out))
	assert.Equal(t, "board/first valid (dry run)\nboard/second valid (dry run)\n", out.String())

	out.Reset()
	require.NoError(t, Apply(context.Background(), nil, read(t, twoBoards), ApplyOptions{DryRun: true, Output: "yaml"}, &out))
	again := read(t, out.String())
	require.Len(t, again, 2)
	assert.Equal(t, "second", again[1].(*insightsv1.Board).Spec.DisplayName, "the dry run prints the objects with defaults filled in")
}

// A malformed List next to a valid board must stop the whole apply, not
// apply the board and skip the List.
func TestReadInputs_MalformedListFailsTheWholeInput(t *testing.T) {
	input := twoBoards + "---\napiVersion: v1\nkind: List\nitmes: []\n"
	_, err := ReadInputs([]string{"-"}, strings.NewReader(input))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "document 3")
}

func TestApply_DryRunPrintsThePayload(t *testing.T) {
	exported := `apiVersion: insights.testkube.io/v1alpha1
kind: Board
metadata:
  name: b
  namespace: testkube
  resourceVersion: "7"
spec: {}
status:
  id: board-uuid
  gitOpsOwner:
    agentId: runner-1
`
	var out bytes.Buffer
	require.NoError(t, Apply(context.Background(), nil, read(t, exported), ApplyOptions{DryRun: true, Output: "yaml"}, &out))
	printed := out.String()
	for _, field := range []string{"namespace", "resourceVersion", "status", "gitOpsOwner"} {
		assert.NotContains(t, printed, field, "a PUT never sends %s", field)
	}
	assert.Contains(t, printed, "name: b")
}

func TestApply_InvalidDocumentSendsNothing(t *testing.T) {
	input := twoBoards + `---
apiVersion: insights.testkube.io/v1alpha1
kind: Board
metadata:
  name: Bad_Slug
spec: {}
`
	c := &fakeClient{}
	err := Apply(context.Background(), c, read(t, input), ApplyOptions{}, &bytes.Buffer{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "board/Bad_Slug")
	assert.Empty(t, c.applied)
}

func TestApply_DuplicateObject(t *testing.T) {
	input := twoBoards + "---\n" + strings.SplitN(twoBoards, "---\n", 2)[0]
	err := Apply(context.Background(), &fakeClient{}, read(t, input), ApplyOptions{}, &bytes.Buffer{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "board/first is declared more than once")
}

func TestApply_UnknownKind(t *testing.T) {
	_, err := ReadInputs([]string{"-"}, strings.NewReader("apiVersion: insights.testkube.io/v1alpha1\nkind: QualityGate\nmetadata:\n  name: x\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown kind "QualityGate"`)

	_, err = ReadInputs([]string{"-"}, strings.NewReader("apiVersion: testworkflows.testkube.io/v1\nkind: TestWorkflow\nmetadata:\n  name: x\n"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not a Control Plane resource")
}

func TestApply_ContinuesPastAFailure(t *testing.T) {
	c := &fakeClient{applyErr: map[string]error{"first": errors.New("managed by GitOps agent prod")}}
	var out bytes.Buffer
	err := Apply(context.Background(), c, read(t, twoBoards), ApplyOptions{}, &out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "board/first: managed by GitOps agent prod")
	assert.Equal(t, "board/second applied\n", out.String())
}

func TestApply_RejectsUnknownOutput(t *testing.T) {
	err := Apply(context.Background(), nil, read(t, twoBoards), ApplyOptions{DryRun: true, Output: "go"}, &bytes.Buffer{})
	require.Error(t, err)
}

func TestReadInputs_Directory(t *testing.T) {
	dir := t.TempDir()
	docs := strings.SplitN(twoBoards, "---\n", 2)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "b.yaml"), []byte(docs[1]), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.yml"), []byte(docs[0]), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("not yaml: ["), 0o600))

	objects, err := ReadInputs([]string{dir}, nil)
	require.NoError(t, err)
	require.Len(t, objects, 2)
	assert.Equal(t, "first", objects[0].GetName(), "files are read in name order")

	_, err = ReadInputs([]string{t.TempDir()}, nil)
	assert.ErrorContains(t, err, "no objects found")
}

func storedBoard(name string) *insightsv1.Board {
	b := &insightsv1.Board{Spec: insightsv1.BoardSpec{DisplayName: strings.ToUpper(name)}}
	b.SetGroupVersionKind(insightsv1.GroupVersion.WithKind(insightsv1.BoardKind))
	b.Name = name
	b.Status = commonv1.ControlPlaneResourceStatus{ID: "id-" + name, CreationSource: commonv1.SourceUI, GitOpsOwner: &commonv1.GitOpsOwner{AgentID: "agent", AgentName: "prod-sync"}}
	return b
}

func boardKind(t *testing.T) declarative.Kind {
	kind, ok := declarative.LookupName("board")
	require.True(t, ok)
	return kind
}

func TestGet(t *testing.T) {
	c := &fakeClient{stored: map[string]declarative.Object{"a": storedBoard("a"), "b": storedBoard("b")}}

	var out bytes.Buffer
	require.NoError(t, Get(context.Background(), c, boardKind(t), "", "pretty", &out))
	assert.Contains(t, out.String(), "prod-sync")
	assert.Contains(t, out.String(), "id-b")

	out.Reset()
	require.NoError(t, Get(context.Background(), c, boardKind(t), "a", "yaml", &out))
	exported := read(t, out.String())
	require.Len(t, exported, 1)
	assert.Equal(t, "A", exported[0].(*insightsv1.Board).Spec.DisplayName)

	assert.ErrorIs(t, Get(context.Background(), c, boardKind(t), "missing", "yaml", &out), client.ErrNotFound)
	assert.Error(t, Get(context.Background(), c, boardKind(t), "a", "go", &out))
}

func TestDelete(t *testing.T) {
	c := &fakeClient{stored: map[string]declarative.Object{"a": storedBoard("a")}}
	var out bytes.Buffer
	require.NoError(t, Delete(context.Background(), c, boardKind(t), "a", false, &out))
	assert.Equal(t, []string{"a"}, c.deleted)
	assert.Equal(t, "board/a deleted\n", out.String())

	assert.ErrorIs(t, Delete(context.Background(), c, boardKind(t), "missing", false, &out), client.ErrNotFound)
	assert.NoError(t, Delete(context.Background(), c, boardKind(t), "missing", true, &out))
}

func TestCommandsPerKind(t *testing.T) {
	gets := NewGetCmds(NewClient)
	require.Len(t, gets, len(declarative.Kinds()))
	assert.Equal(t, "board [name]", gets[0].Use)
	assert.Equal(t, []string{"boards"}, gets[0].Aliases)
	assert.Equal(t, "board <name>", NewDeleteCmds(NewClient)[0].Use)
}

func TestCommandsUseTheirClientFactory(t *testing.T) {
	c := &fakeClient{stored: map[string]declarative.Object{"a": storedBoard("a")}}
	factory := func(*cobra.Command) (client.Client, error) { return c, nil }

	apply := NewApplyCmd(factory)
	apply.SetIn(strings.NewReader(twoBoards))
	apply.SetOut(&bytes.Buffer{})
	apply.SetArgs([]string{"-f", "-"})
	require.NoError(t, apply.Execute())
	assert.Len(t, c.applied, 2)

	del := NewDeleteCmds(factory)[0]
	del.SetOut(&bytes.Buffer{})
	del.SetArgs([]string{"a"})
	require.NoError(t, del.Execute())
	assert.Equal(t, []string{"a"}, c.deleted)
}
