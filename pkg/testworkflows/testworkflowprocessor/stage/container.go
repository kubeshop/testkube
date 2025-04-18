package stage

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/expressions"
	"github.com/kubeshop/testkube/pkg/imageinspector"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowresolver"
)

type container struct {
	parent  *container
	toolkit bool
	Cr      testworkflowsv1.ContainerConfig `expr:"include"`
}

type ContainerComposition interface {
	Root() Container
	Parent() Container
	CreateChild() Container

	Resolve(m ...expressions.Machine) error
}

type ContainerAccessors interface {
	Env() []testworkflowsv1.EnvVar
	EnvFrom() []corev1.EnvFromSource
	VolumeMounts() []corev1.VolumeMount

	ImagePullPolicy() corev1.PullPolicy
	Image() string
	Command() []string
	Args() []string
	WorkingDir() string

	Detach() Container
	ToKubernetesTemplate() (corev1.Container, error)

	Resources() testworkflowsv1.Resources
	SecurityContext() *corev1.SecurityContext

	HasVolumeAt(path string) bool
	ToContainerConfig() testworkflowsv1.ContainerConfig
	IsToolkit() bool
	NeedsImageData(isGroupNeeded bool) bool
}

type ContainerMutations[T any] interface {
	AppendEnv(env ...corev1.EnvVar) T
	AppendEnvMap(env map[string]string) T
	AppendEnvFrom(envFrom ...corev1.EnvFromSource) T
	AppendVolumeMounts(volumeMounts ...corev1.VolumeMount) T
	SetImagePullPolicy(policy corev1.PullPolicy) T
	SetImage(image string) T
	SetCommand(command ...string) T
	SetArgs(args ...string) T
	SetWorkingDir(workingDir string) T // "" = default to the image
	SetResources(resources testworkflowsv1.Resources) T
	SetSecurityContext(sc *corev1.SecurityContext) T

	ApplyCR(cr *testworkflowsv1.ContainerConfig) T
	ApplyImageData(image *imageinspector.Info, resolvedImageName string) error
	EnableToolkit(ref string) T
}

//go:generate mockgen -destination=./mock_container.go -package=stage "github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage" Container
type Container interface {
	ContainerComposition
	ContainerAccessors
	ContainerMutations[Container]
}

func NewContainer() Container {
	return &container{}
}

func sum[T any](s1 []T, s2 []T) []T {
	if len(s1) == 0 {
		return s2
	}
	if len(s2) == 0 {
		return s1
	}
	return append(append(make([]T, 0, len(s1)+len(s2)), s1...), s2...)
}

// Composition

func (c *container) Root() Container {
	if c.parent == nil {
		return c
	}
	return c.parent.Root()
}

func (c *container) Parent() Container {
	return c.parent
}

func (c *container) CreateChild() Container {
	return &container{parent: c, toolkit: c.toolkit}
}

// Getters

func (c *container) Env() []testworkflowsv1.EnvVar {
	if c.parent == nil {
		return c.Cr.Env
	}
	return sum(c.parent.Env(), c.Cr.Env)
}

func (c *container) EnvFrom() []corev1.EnvFromSource {
	if c.parent == nil {
		return c.Cr.EnvFrom
	}
	return sum(c.parent.EnvFrom(), c.Cr.EnvFrom)
}

func (c *container) VolumeMounts() []corev1.VolumeMount {
	if c.parent == nil {
		return c.Cr.VolumeMounts
	}
	return sum(c.parent.VolumeMounts(), c.Cr.VolumeMounts)
}

func (c *container) ImagePullPolicy() corev1.PullPolicy {
	if c.parent == nil || c.Cr.ImagePullPolicy != "" {
		return c.Cr.ImagePullPolicy
	}
	return c.parent.ImagePullPolicy()
}

func (c *container) Image() string {
	if c.parent == nil || c.Cr.Image != "" {
		return c.Cr.Image
	}
	return c.parent.Image()
}

func (c *container) Command() []string {
	// Do not inherit command, if the Image was replaced on this depth
	if c.parent == nil || c.Cr.Command != nil || (c.Cr.Image != "" && c.Cr.Image != c.Image()) {
		if c.Cr.Command == nil {
			return nil
		}
		return *c.Cr.Command
	}
	return c.parent.Command()
}

func (c *container) Args() []string {
	// Do not inherit args, if the Image or Command was replaced on this depth
	if c.parent == nil || (c.Cr.Args != nil && len(*c.Cr.Args) > 0) || c.Cr.Command != nil || (c.Cr.Image != "" && c.Cr.Image != c.Image()) {
		if c.Cr.Args == nil {
			return nil
		}
		return *c.Cr.Args
	}
	return c.parent.Args()
}

func (c *container) WorkingDir() string {
	path := ""
	if c.Cr.WorkingDir != nil {
		path = *c.Cr.WorkingDir
	}
	if c.parent == nil {
		return path
	}
	firstParam := c.parent.WorkingDir()
	secondParam := path
	if strings.HasPrefix(firstParam, "{{") && strings.HasSuffix(firstParam, "}}") {
		firstParam = strings.TrimSuffix(strings.TrimPrefix(firstParam, "{{"), "}}")
	} else {
		firstParam = fmt.Sprintf("%q", firstParam)
	}
	if strings.HasPrefix(secondParam, "{{") && strings.HasSuffix(secondParam, "}}") {
		secondParam = strings.TrimSuffix(strings.TrimPrefix(secondParam, "{{"), "}}")
	} else {
		secondParam = fmt.Sprintf("%q", secondParam)
	}
	return fmt.Sprintf("{{makepath(%s, %s)}}", firstParam, secondParam)
}

func (c *container) Resources() (r testworkflowsv1.Resources) {
	if c.parent != nil {
		r = *common.Ptr(c.parent.Resources()).DeepCopy()
	}
	if c.Cr.Resources == nil {
		return
	}
	if len(c.Cr.Resources.Requests) > 0 {
		r.Requests = c.Cr.Resources.Requests
	}
	if len(c.Cr.Resources.Limits) > 0 {
		r.Limits = c.Cr.Resources.Limits
	}
	return
}

func (c *container) SecurityContext() *corev1.SecurityContext {
	if c.parent == nil || c.parent.SecurityContext() == nil {
		return c.Cr.SecurityContext
	}
	if c.Cr.SecurityContext == nil {
		return c.parent.SecurityContext()
	}
	return testworkflowresolver.MergeSecurityContext(c.parent.SecurityContext().DeepCopy(), c.Cr.SecurityContext)
}

func (c *container) HasVolumeAt(path string) bool {
	absPath := path
	if !filepath.IsAbs(path) {
		absPath = filepath.Join(c.WorkingDir(), path)
	}
	mounts := c.VolumeMounts()
	absPath = filepath.Clean(absPath)
	for _, mount := range mounts {
		mountPath := filepath.Clean(mount.MountPath)
		if absPath == mountPath || strings.HasPrefix(absPath, mountPath+"/") {
			return true
		}
	}
	return false
}

// Mutations

func (c *container) AppendEnv(env ...corev1.EnvVar) Container {
	needsDedupe := false
	for i := range env {
		if testworkflowresolver.HasEnvVar(c.Cr.Env, env[i].Name) {
			needsDedupe = true
			break
		}
	}
	c.Cr.Env = append(c.Cr.Env, common.MapSlice(env, func(e corev1.EnvVar) testworkflowsv1.EnvVar {
		return testworkflowsv1.EnvVar{EnvVar: e}
	})...)
	if needsDedupe {
		c.Cr.Env = testworkflowresolver.DedupeEnvVars(c.Cr.Env)
	}
	return c
}

func (c *container) AppendEnvMap(env map[string]string) Container {
	for k, v := range env {
		c.Cr.Env = append(c.Cr.Env, testworkflowsv1.EnvVar{EnvVar: corev1.EnvVar{Name: k, Value: v}})
	}
	return c
}

func (c *container) AppendEnvFrom(envFrom ...corev1.EnvFromSource) Container {
	c.Cr.EnvFrom = append(c.Cr.EnvFrom, envFrom...)
	return c
}

func (c *container) AppendVolumeMounts(volumeMounts ...corev1.VolumeMount) Container {
	c.Cr.VolumeMounts = append(c.Cr.VolumeMounts, volumeMounts...)
	return c
}

func (c *container) SetImagePullPolicy(policy corev1.PullPolicy) Container {
	c.Cr.ImagePullPolicy = policy
	return c
}

func (c *container) SetImage(image string) Container {
	c.Cr.Image = image
	return c
}

func (c *container) SetCommand(command ...string) Container {
	c.Cr.Command = &command
	return c
}

func (c *container) SetArgs(args ...string) Container {
	c.Cr.Args = &args
	return c
}

func (c *container) SetWorkingDir(workingDir string) Container {
	c.Cr.WorkingDir = &workingDir
	return c
}

func (c *container) SetResources(resources testworkflowsv1.Resources) Container {
	c.Cr.Resources = &resources
	return c
}

func (c *container) SetSecurityContext(sc *corev1.SecurityContext) Container {
	c.Cr.SecurityContext = sc
	return c
}

func (c *container) ApplyCR(config *testworkflowsv1.ContainerConfig) Container {
	c.Cr = *testworkflowresolver.MergeContainerConfig(&c.Cr, config)
	return c
}

func (c *container) ToContainerConfig() testworkflowsv1.ContainerConfig {
	env := testworkflowresolver.DedupeEnvVars(slices.Clone(c.Env()))
	for i := range env {
		env[i] = *env[i].DeepCopy()
	}
	envFrom := slices.Clone(c.EnvFrom())
	for i := range envFrom {
		envFrom[i] = *envFrom[i].DeepCopy()
	}
	volumeMounts := slices.Clone(c.VolumeMounts())
	for i := range volumeMounts {
		volumeMounts[i] = *volumeMounts[i].DeepCopy()
	}
	workingDir := common.Ptr(c.WorkingDir())
	if *workingDir == "" {
		workingDir = nil
	}
	resources := &testworkflowsv1.Resources{
		Requests: maps.Clone(c.Resources().Requests),
		Limits:   maps.Clone(c.Resources().Limits),
	}
	if len(resources.Requests) == 0 && len(resources.Limits) == 0 {
		resources = nil
	}
	args := common.Ptr(slices.Clone(c.Args()))
	if *args == nil {
		args = nil
	}
	command := common.Ptr(slices.Clone(c.Command()))
	if *command == nil {
		command = nil
	}
	return testworkflowsv1.ContainerConfig{
		WorkingDir:      workingDir,
		Image:           c.Image(),
		ImagePullPolicy: c.ImagePullPolicy(),
		Env:             env,
		EnvFrom:         envFrom,
		Command:         command,
		Args:            args,
		Resources:       resources,
		SecurityContext: c.SecurityContext().DeepCopy(),
		VolumeMounts:    volumeMounts,
	}
}

func (c *container) Detach() Container {
	c.toolkit = c.IsToolkit()
	c.Cr = c.ToContainerConfig()
	c.parent = nil
	return c
}

func (c *container) ToKubernetesTemplate() (corev1.Container, error) {
	cr := c.ToContainerConfig()
	var command []string
	if cr.Command != nil {
		command = *cr.Command
	}
	var args []string
	if cr.Args != nil {
		args = *cr.Args
	}
	workingDir := ""
	if cr.WorkingDir != nil {
		workingDir = *cr.WorkingDir
	}
	resources, resourcesErr := MapResourcesToKubernetesResources(cr.Resources)
	if resourcesErr != nil {
		return corev1.Container{}, resourcesErr
	}
	return corev1.Container{
		Image:           cr.Image,
		ImagePullPolicy: cr.ImagePullPolicy,
		Command:         command,
		Args:            args,
		Env: common.MapSlice(cr.Env, func(e testworkflowsv1.EnvVar) corev1.EnvVar {
			return corev1.EnvVar{Name: e.Name, Value: e.Value, ValueFrom: e.ValueFrom}
		}),
		EnvFrom:         cr.EnvFrom,
		VolumeMounts:    cr.VolumeMounts,
		Resources:       resources,
		WorkingDir:      workingDir,
		SecurityContext: cr.SecurityContext,
	}, nil
}

func (c *container) ApplyImageData(image *imageinspector.Info, resolvedImageName string) error {
	if resolvedImageName != "" && c.Image() != resolvedImageName {
		c.SetImage(resolvedImageName)
	}
	if image == nil {
		return nil
	}
	err := c.Resolve(expressions.NewMachine().
		Register("image.command", image.Entrypoint).
		Register("image.args", image.Cmd).
		Register("image.workingDir", image.WorkingDir))
	if err != nil {
		return err
	}
	command := c.Command()
	args := c.Args()
	if len(command) == 0 {
		c.SetCommand(image.Entrypoint...)
		if len(args) == 0 {
			c.SetArgs(image.Cmd...)
		} else {
			c.SetArgs(args...)
		}
	}
	if image.WorkingDir != "" && c.WorkingDir() == "" {
		c.SetWorkingDir(image.WorkingDir)
	}
	return nil
}

// NeedsImageData checks if we need to fetch metadata of the destination image from Container Registry
func (c *container) NeedsImageData(isGroupNeeded bool) bool {
	if len(c.Command()) == 0 || c.WorkingDir() == "" {
		return true
	}
	securityContext := c.SecurityContext()
	if isGroupNeeded && (securityContext == nil || securityContext.RunAsGroup == nil) {
		return true
	}
	usesVariables := false
	expressions.WalkVariables(c, func(name string) error {
		if name == "image.command" || name == "image.args" || name == "image.workingDir" {
			usesVariables = true
			return expressions.ErrWalkStop
		}
		return nil
	})
	return usesVariables
}

func (c *container) IsToolkit() bool {
	return c.toolkit || (c.parent != nil && c.parent.IsToolkit()) || slices.Contains(c.Cr.Env, BypassToolkitCheck)
}

func (c *container) MarkAsToolkit() Container {
	c.toolkit = true
	return c
}

func (c *container) EnableToolkit(ref string) Container {
	return c.
		MarkAsToolkit().
		AppendEnv(corev1.EnvVar{
			Name:      "TK_IP",
			ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "status.podIP"}},
		}).
		AppendEnv(corev1.EnvVar{
			Name:  "TK_REF",
			Value: ref,
		})
}

func (c *container) envMachine() expressions.Machine {
	return expressions.NewMachine().
		RegisterAccessor(func(name string) (interface{}, bool) {
			if !strings.HasPrefix(name, "env.") {
				return nil, false
			}
			env := c.Env()
			name = name[4:]
			for i := len(env) - 1; i >= 0; i-- {
				if env[i].Name == name && env[i].ValueFrom == nil {
					value, err := expressions.EvalTemplate(env[i].Value)
					if err == nil {
						return value, true
					}
					break
				}
			}
			return nil, false
		})
}

func (c *container) Resolve(m ...expressions.Machine) error {
	return expressions.Simplify(c, append([]expressions.Machine{c.envMachine()}, m...)...)
}
