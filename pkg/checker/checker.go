package checker

import (
	"context"
	"fmt"

	"github.com/kubeshop/testkube/pkg/k8sclient"
	"k8s.io/client-go/kubernetes"
)

const (
	checkPass = "Passed"
	checkFail = "Failed"

	apiServerDeploymentSelector = "app.kubernetes.io/name=api-server"
	operatorDeploymentSelector  = "control-plane=controller-manager"

	ClusterCheck            = "Cluster Check"
	TestkubeAPICheck        = "Testkube API Check"
	TestkubePermissionCheck = "Testkube Permission Check"
)

// essential checks will run even if ignore-blocker is true for any check
// belonging to a checksuite which needs to be success before any other checksuite's checks
var essentialChecks = map[string]bool{
	ClusterCheck: true,
}

type Checker struct {
	Description string                      // description of the check
	Blocker     bool                        // subsequent checks blocked if this check fails
	Run         func(context.Context) error // executable function for the check
}

type CheckSuiteName string

type CheckSuite struct {
	Name   CheckSuiteName // name of the check
	Checks []Checker      // list of checks to be performed
	Enable bool           // if true, suite will be executed
}

type SystemChecker struct {
	Suites       []CheckSuite // list of check suites
	SuiteResults []CheckSuiteOutput
	Client       *kubernetes.Clientset // k8s client
}

func NewSystemChecker(enabledSuites []CheckSuiteName) *SystemChecker {
	sc := &SystemChecker{}
	sc.Suites = sc.loadAllSuites()

	enabled := make(map[CheckSuiteName]bool)
	for _, name := range enabledSuites {
		enabled[name] = true
	}

	for i, suite := range sc.Suites {
		if enabled[suite.Name] {
			sc.Suites[i].Enable = true
		}
	}

	return sc
}

func (sc *SystemChecker) ExecuteSuite(ignoreBlocker bool) bool {
	ctx := context.Background()
	allSuccess := true
	stopExecution := false
	checkSuiteOutputs := []CheckSuiteOutput{}
	for _, suite := range sc.Suites {
		if !suite.Enable {
			continue
		}
		checkResults := []CheckResult{}
		for _, check := range suite.Checks {
			err := check.Run(ctx)
			checkOutput := CheckResult{
				Description: check.Description,
			}

			if err != nil {
				checkOutput.Error = err.Error()
				checkOutput.Result = checkFail
				allSuccess = false
				if check.Blocker && !ignoreBlocker || essentialChecks[string(suite.Name)] {
					stopExecution = true
				}
			} else {
				checkOutput.Result = checkPass
			}

			checkResults = append(checkResults, checkOutput)
			if stopExecution {
				break
			}
		}
		checkSuiteOutputs = append(checkSuiteOutputs, CheckSuiteOutput{
			CheckSuiteName: suite.Name,
			CheckResults:   checkResults,
		})
		sc.SuiteResults = checkSuiteOutputs
		if stopExecution {
			break
		}
	}
	return allSuccess
}

func (sc *SystemChecker) loadAllSuites() []CheckSuite {
	return []CheckSuite{
		{
			Name: ClusterCheck,
			Checks: []Checker{
				{
					Description: "Check if cluster is reachable",
					Run: func(ctx context.Context) error {
						clientSet, err := k8sclient.ConnectToK8s()
						if err != nil {
							return fmt.Errorf("%s", err.Error())
						}
						sc.Client = clientSet
						return nil
					},
					Blocker: true,
				},
				{
					Description: "Check if Kubernetes version is compatible",
					Run: func(ctx context.Context) error {
						version, err := k8sclient.GetClusterVersion(sc.Client)
						if err != nil {
							return err
						}
						return k8sclient.IsRunningMinKubeVersion(version)
					},
					Blocker: true,
				},
			},
		},
		{
			Name: TestkubeAPICheck,
			Checks: []Checker{
				{
					Description: "Check if Testkube API is reachable",
					Run: func(ctx context.Context) error {
						isReady, err := k8sclient.CheckDeploymentReady(ctx, sc.Client, "testkube", apiServerDeploymentSelector)
						if err != nil {
							return err
						}
						if !isReady {
							return fmt.Errorf("Testkube api-server deployment not ready")
						}
						return nil
					},
					Blocker: true,
				},
				{
					Description: "Check if Testkube Operator is reachable",
					Run: func(ctx context.Context) error {
						isReady, err := k8sclient.CheckDeploymentReady(ctx, sc.Client, "testkube", operatorDeploymentSelector)
						if err != nil {
							return err
						}
						if !isReady {
							return fmt.Errorf("Testkube operator deployment not ready")
						}
						return nil
					},
					Blocker: false,
				},
			},
		},
		{
			Name: TestkubePermissionCheck,
			Checks: []Checker{
				{
					Description: "Check if Testkube has necessary permissions",
					Run: func(ctx context.Context) error {
						expectedBindings := []string{
							"watchers-rb-testkube",
						}
						_, err := k8sclient.CheckClusterRoleBindingExists(ctx, sc.Client, expectedBindings)
						return err
					},
					Blocker: false,
				},
			},
		},
	}
}
