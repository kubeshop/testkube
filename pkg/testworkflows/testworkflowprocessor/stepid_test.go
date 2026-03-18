package testworkflowprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
)

func TestValidateStepId(t *testing.T) {
	tests := map[string]struct {
		id      string
		wantErr bool
	}{
		"valid simple":              {id: "auth", wantErr: false},
		"valid with number":         {id: "step1", wantErr: false},
		"valid snake_case":          {id: "get_auth_token", wantErr: false},
		"valid with underscores":    {id: "my__id", wantErr: false},
		"valid trailing underscore": {id: "my_id_", wantErr: false},
		"valid former reserved":     {id: "config", wantErr: false},
		"invalid uppercase":         {id: "Auth", wantErr: true},
		"invalid hyphen":            {id: "get-auth", wantErr: true},
		"invalid starts num":        {id: "1step", wantErr: true},
		"invalid empty":             {id: "", wantErr: true},
		"invalid spaces":            {id: "my step", wantErr: true},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := ValidateStepId(tc.id)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDeriveStepId(t *testing.T) {
	tests := map[string]struct {
		name string
		want string
	}{
		"simple":             {name: "Build", want: "build"},
		"multi word":         {name: "Run Load Test", want: "run_load_test"},
		"with parens":        {name: "Get Auth Token (v2)", want: "get_auth_token_v2"},
		"camelCase":          {name: "getNodeCount", want: "getnodecount"},
		"already snake_case": {name: "run_tests", want: "run_tests"},
		"empty":              {name: "", want: ""},
		"special chars only": {name: "---", want: ""},
		"starts with number": {name: "1st step", want: ""},
		"trailing space":     {name: "test ", want: "test"},
		"multiple spaces":    {name: "run  load  test", want: "run_load_test"},
		"hyphens replaced":   {name: "get-auth-token", want: "get_auth_token"},
		"unicode name":       {name: "Résumé", want: "résumé"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := DeriveStepId(tc.name)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestResolveAndValidateStepIds(t *testing.T) {
	t.Run("valid explicit ids", func(t *testing.T) {
		spec := &testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepMeta: testworkflowsv1.StepMeta{Id: "build", Name: "Build"}},
				{StepMeta: testworkflowsv1.StepMeta{Id: "test", Name: "Test"}},
			},
		}
		assert.NoError(t, ResolveAndValidateStepIds(spec))
	})

	t.Run("duplicate explicit ids rejected", func(t *testing.T) {
		spec := &testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepMeta: testworkflowsv1.StepMeta{Id: "build", Name: "Build"}},
				{StepMeta: testworkflowsv1.StepMeta{Id: "build", Name: "Build 2"}},
			},
		}
		assert.Error(t, ResolveAndValidateStepIds(spec))
	})

	t.Run("invalid format rejected", func(t *testing.T) {
		spec := &testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepMeta: testworkflowsv1.StepMeta{Id: "My-Step"}},
			},
		}
		assert.Error(t, ResolveAndValidateStepIds(spec))
	})

	t.Run("auto-derived ids set on steps", func(t *testing.T) {
		spec := &testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepMeta: testworkflowsv1.StepMeta{Name: "Build Binary"}},
				{StepMeta: testworkflowsv1.StepMeta{Name: "Run Tests"}},
			},
		}
		assert.NoError(t, ResolveAndValidateStepIds(spec))
		assert.Equal(t, "build_binary", spec.Steps[0].Id)
		assert.Equal(t, "run_tests", spec.Steps[1].Id)
	})

	t.Run("auto-derived conflict gets index suffix", func(t *testing.T) {
		spec := &testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepMeta: testworkflowsv1.StepMeta{Name: "Build"}},
				{StepMeta: testworkflowsv1.StepMeta{Name: "Build"}},
				{StepMeta: testworkflowsv1.StepMeta{Name: "Build"}},
			},
		}
		assert.NoError(t, ResolveAndValidateStepIds(spec))
		assert.Equal(t, "build", spec.Steps[0].Id)
		assert.Equal(t, "build_1", spec.Steps[1].Id)
		assert.Equal(t, "build_2", spec.Steps[2].Id)
	})

	t.Run("no ids or names is valid", func(t *testing.T) {
		spec := &testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepOperations: testworkflowsv1.StepOperations{Shell: "echo hello"}},
			},
		}
		assert.NoError(t, ResolveAndValidateStepIds(spec))
	})

	t.Run("nested duplicate ids rejected", func(t *testing.T) {
		spec := &testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{
					StepMeta: testworkflowsv1.StepMeta{Id: "parent", Name: "Parent"},
					Steps: []testworkflowsv1.Step{
						{StepMeta: testworkflowsv1.StepMeta{Id: "parent"}},
					},
				},
			},
		}
		assert.Error(t, ResolveAndValidateStepIds(spec))
	})

	t.Run("unicode name not auto-derived", func(t *testing.T) {
		spec := &testworkflowsv1.TestWorkflowSpec{
			Steps: []testworkflowsv1.Step{
				{StepMeta: testworkflowsv1.StepMeta{Name: "Résumé"}},
			},
		}
		assert.NoError(t, ResolveAndValidateStepIds(spec))
		assert.Equal(t, "", spec.Steps[0].Id)
	})

	t.Run("cross-section uniqueness enforced", func(t *testing.T) {
		spec := &testworkflowsv1.TestWorkflowSpec{
			Setup: []testworkflowsv1.Step{
				{StepMeta: testworkflowsv1.StepMeta{Id: "init", Name: "Init"}},
			},
			Steps: []testworkflowsv1.Step{
				{StepMeta: testworkflowsv1.StepMeta{Id: "init", Name: "Init 2"}},
			},
		}
		assert.Error(t, ResolveAndValidateStepIds(spec))
	})
}
