package testworkflowexecutiontelemetry

import (
	"context"
	"testing"

	gomock "go.uber.org/mock/gomock"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	configRepo "github.com/kubeshop/testkube/pkg/repository/config"
)

func Test_apiTCL_getClusterID(t *testing.T) {

	t.Run("Get Cluster ID", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		configRepo := configRepo.NewMockRepository(ctrl)
		clusterID := "cluster-id"
		configRepo.EXPECT().GetUniqueClusterId(gomock.Any()).Return(clusterID, nil)
		if got := GetClusterID(context.Background(), configRepo); got != clusterID {
			t.Errorf("apiTCL.getClusterID() = %v, want %v", got, clusterID)
		}
	})
}

func Test_GetImage(t *testing.T) {

	t.Run("Get Image from empty container", func(t *testing.T) {
		if got := GetImage(nil); got != "" {
			t.Errorf("getImage() = %v, wanted empty", got)
		}
	})
	t.Run("Get Image from container", func(t *testing.T) {
		image := "container-image"
		container := &testworkflowsv1.ContainerConfig{
			Image: image,
		}

		if got := GetImage(container); got != image {
			t.Errorf("getImage() = %v, want %v", got, image)
		}
	})
}

func Test_HasArtifacts(t *testing.T) {
	type args struct {
		steps []testworkflowsv1.Step
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "No artifacts",
			args: args{
				steps: []testworkflowsv1.Step{
					{
						Use:      []testworkflowsv1.TemplateRef{},
						Template: &testworkflowsv1.TemplateRef{},
						Setup:    []testworkflowsv1.Step{},
						Steps:    []testworkflowsv1.Step{},
					},
				},
			},
			want: false,
		},
		{
			name: "Has artifacts on first level only",
			args: args{
				steps: []testworkflowsv1.Step{
					{
						StepOperations: testworkflowsv1.StepOperations{
							Artifacts: &testworkflowsv1.StepArtifacts{
								Paths: []string{"path"},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "Has artifacts on third level",
			args: args{
				steps: []testworkflowsv1.Step{
					{
						Setup: []testworkflowsv1.Step{
							{
								Setup: []testworkflowsv1.Step{
									{
										StepOperations: testworkflowsv1.StepOperations{
											Artifacts: &testworkflowsv1.StepArtifacts{
												Paths: []string{"path"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "Has artifacts on multiple levels",
			args: args{
				steps: []testworkflowsv1.Step{
					{
						StepOperations: testworkflowsv1.StepOperations{
							Artifacts: &testworkflowsv1.StepArtifacts{
								Paths: []string{"path"},
							},
						},
						Setup: []testworkflowsv1.Step{
							{
								StepOperations: testworkflowsv1.StepOperations{
									Artifacts: &testworkflowsv1.StepArtifacts{
										Paths: []string{"path"},
									},
								},
								Setup: []testworkflowsv1.Step{
									{
										StepOperations: testworkflowsv1.StepOperations{
											Artifacts: &testworkflowsv1.StepArtifacts{
												Paths: []string{"path"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasStepLike(tt.args.steps, HasArtifacts); got != tt.want {
				t.Errorf("hasArtifacts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_HasTemplateArtifacts(t *testing.T) {
	type args struct {
		steps []testworkflowsv1.IndependentStep
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "No artifacts",
			args: args{
				steps: []testworkflowsv1.IndependentStep{
					{
						Setup: []testworkflowsv1.IndependentStep{},
						Steps: []testworkflowsv1.IndependentStep{},
					},
				},
			},
			want: false,
		},
		{
			name: "Has artifacts on first level only",
			args: args{
				steps: []testworkflowsv1.IndependentStep{
					{
						StepOperations: testworkflowsv1.StepOperations{
							Artifacts: &testworkflowsv1.StepArtifacts{
								Paths: []string{"path"},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "Has artifacts on third level",
			args: args{
				steps: []testworkflowsv1.IndependentStep{
					{
						Setup: []testworkflowsv1.IndependentStep{
							{
								Setup: []testworkflowsv1.IndependentStep{
									{
										StepOperations: testworkflowsv1.StepOperations{
											Artifacts: &testworkflowsv1.StepArtifacts{
												Paths: []string{"path"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "Has artifacts on multiple levels",
			args: args{
				steps: []testworkflowsv1.IndependentStep{
					{
						StepOperations: testworkflowsv1.StepOperations{
							Artifacts: &testworkflowsv1.StepArtifacts{
								Paths: []string{"path"},
							},
						},
						Setup: []testworkflowsv1.IndependentStep{
							{
								StepOperations: testworkflowsv1.StepOperations{
									Artifacts: &testworkflowsv1.StepArtifacts{
										Paths: []string{"path"},
									},
								},
								Setup: []testworkflowsv1.IndependentStep{
									{
										StepOperations: testworkflowsv1.StepOperations{
											Artifacts: &testworkflowsv1.StepArtifacts{
												Paths: []string{"path"},
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasIndependentStepLike(tt.args.steps, HasTemplateArtifacts); got != tt.want {
				t.Errorf("hasArtifacts() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_HasKubeshopGitURI(t *testing.T) {
	type args struct {
		spec testworkflowsv1.TestWorkflowSpec
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "No Kubeshop Git URI",
			args: args{
				spec: testworkflowsv1.TestWorkflowSpec{
					TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
						Content: &testworkflowsv1.Content{
							Git: &testworkflowsv1.ContentGit{
								Uri: "test-uri",
							},
						},
					},
				},
			},
			want: false,
		},
		{
			name: "Has Kubeshop URI on first level only",
			args: args{
				spec: testworkflowsv1.TestWorkflowSpec{
					TestWorkflowSpecBase: testworkflowsv1.TestWorkflowSpecBase{
						Content: &testworkflowsv1.Content{
							Git: &testworkflowsv1.ContentGit{
								Uri: "github.com/kubeshop/testkube-tests-uri",
							},
						},
					},
				},
			},
			want: true,
		},
		{
			name: "Has Kubeshop URI on third level only",
			args: args{
				spec: testworkflowsv1.TestWorkflowSpec{
					Steps: []testworkflowsv1.Step{
						{
							Setup: []testworkflowsv1.Step{
								{
									StepSource: testworkflowsv1.StepSource{
										Content: &testworkflowsv1.Content{
											Git: &testworkflowsv1.ContentGit{
												Uri: "github.com/kubeshop/testkube-tests-uri",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsKubeshopGitURI(tt.args.spec.Content) || HasWorkflowStepLike(tt.args.spec, HasKubeshopGitURI); got != tt.want {
				t.Errorf("hasKubeshopGitURI() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_GetDataSource(t *testing.T) {
	type args struct {
		content *testworkflowsv1.Content
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Empty content",
			args: args{
				content: nil,
			},
			want: "",
		},
		{
			name: "Git data source",
			args: args{
				content: &testworkflowsv1.Content{
					Git: &testworkflowsv1.ContentGit{
						Uri: "test-uri",
					},
				},
			},
			want: "git",
		},
		{
			name: "Files data source",
			args: args{
				content: &testworkflowsv1.Content{
					Files: []testworkflowsv1.ContentFile{
						{
							Path: "test-path",
						},
					},
				},
			},
			want: "files",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetDataSource(tt.args.content); got != tt.want {
				t.Errorf("getDataSource() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_triggeredByBucket(t *testing.T) {
	type args struct {
		actorType     string
		interfaceType string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Cron actor",
			args: args{actorType: "cron"},
			want: "scheduled",
		},
		{
			name: "TestTrigger actor",
			args: args{actorType: "testtrigger"},
			want: "event-triggered",
		},
		{
			name: "Workflow chained from another workflow",
			args: args{actorType: "testworkflow"},
			want: "composite",
		},
		{
			name: "Workflow chained from another execution",
			args: args{actorType: "testworkflowexecution"},
			want: "composite",
		},
		{
			name: "Program actor",
			args: args{actorType: "program"},
			want: "automation",
		},
		{
			name: "User via CI/CD interface",
			args: args{actorType: "user", interfaceType: "ci/cd"},
			want: "ci-cd",
		},
		{
			name: "User via UI",
			args: args{actorType: "user", interfaceType: "ui"},
			want: "human-ui",
		},
		{
			name: "User via CLI",
			args: args{actorType: "user", interfaceType: "cli"},
			want: "human-cli",
		},
		{
			name: "User via API",
			args: args{actorType: "user", interfaceType: "api"},
			want: "api",
		},
		{
			name: "User with internal interface falls back to unknown",
			args: args{actorType: "user", interfaceType: "internal"},
			want: "unknown",
		},
		{
			name: "User with no interface falls back to unknown",
			args: args{actorType: "user"},
			want: "unknown",
		},
		{
			name: "Empty actor and interface",
			args: args{},
			want: "unknown",
		},
		{
			name: "Unrecognised actor",
			args: args{actorType: "made-up", interfaceType: "cli"},
			want: "unknown",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := triggeredByBucket(tt.args.actorType, tt.args.interfaceType); got != tt.want {
				t.Errorf("triggeredByBucket(%q, %q) = %v, want %v", tt.args.actorType, tt.args.interfaceType, got, tt.want)
			}
		})
	}
}
