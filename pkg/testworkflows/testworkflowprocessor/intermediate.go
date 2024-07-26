package testworkflowprocessor

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowresolver"
)

//go:generate mockgen -destination=./mock_intermediate.go -package=testworkflowprocessor "github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor" Intermediate
type Intermediate interface {
	RefCounter

	ContainerDefaults() stage.Container
	PodConfig() testworkflowsv1.PodConfig
	JobConfig() testworkflowsv1.JobConfig

	ConfigMaps() []corev1.ConfigMap
	Secrets() []corev1.Secret
	Volumes() []corev1.Volume

	AppendJobConfig(cfg *testworkflowsv1.JobConfig) Intermediate
	AppendPodConfig(cfg *testworkflowsv1.PodConfig) Intermediate

	AddConfigMap(configMap corev1.ConfigMap) Intermediate
	AddSecret(secret corev1.Secret) Intermediate
	AddVolume(volume corev1.Volume) Intermediate

	AddEmptyDirVolume(source *corev1.EmptyDirVolumeSource, mountPath string) corev1.VolumeMount

	AddTextFile(file string, mode *int32) (corev1.VolumeMount, error)
	AddBinaryFile(file []byte, mode *int32) (corev1.VolumeMount, error)
}

type intermediate struct {
	RefCounter

	// Routine
	Root      stage.GroupStage `expr:"include"`
	Container stage.Container  `expr:"include"`

	// Job & Pod resources & data
	Pod testworkflowsv1.PodConfig `expr:"include"`
	Job testworkflowsv1.JobConfig `expr:"include"`

	// Actual Kubernetes resources to use
	Secs []corev1.Secret    `expr:"force"`
	Cfgs []corev1.ConfigMap `expr:"force"`

	// Storing files
	Files ConfigMapFiles `expr:"include"`
}

func NewIntermediate() Intermediate {
	ref := NewRefCounter()
	return &intermediate{
		RefCounter: ref,
		Root:       stage.NewGroupStage("", true),
		Container:  stage.NewContainer(),
		Files:      NewConfigMapFiles(fmt.Sprintf("{{resource.id}}-%s", ref.NextRef()), nil)}
}

func (s *intermediate) ContainerDefaults() stage.Container {
	return s.Container
}

func (s *intermediate) JobConfig() testworkflowsv1.JobConfig {
	return s.Job
}

func (s *intermediate) PodConfig() testworkflowsv1.PodConfig {
	return s.Pod
}

func (s *intermediate) ConfigMaps() []corev1.ConfigMap {
	return append(s.Cfgs, s.Files.ConfigMaps()...)
}

func (s *intermediate) Secrets() []corev1.Secret {
	return s.Secs
}

func (s *intermediate) Volumes() []corev1.Volume {
	return append(s.Pod.Volumes, s.Files.Volumes()...)
}

func (s *intermediate) AppendJobConfig(cfg *testworkflowsv1.JobConfig) Intermediate {
	s.Job = *testworkflowresolver.MergeJobConfig(&s.Job, cfg)
	return s
}

func (s *intermediate) AppendPodConfig(cfg *testworkflowsv1.PodConfig) Intermediate {
	s.Pod = *testworkflowresolver.MergePodConfig(&s.Pod, cfg)
	return s
}

func (s *intermediate) AddVolume(volume corev1.Volume) Intermediate {
	s.Pod.Volumes = append(s.Pod.Volumes, volume)
	return s
}

func (s *intermediate) AddConfigMap(configMap corev1.ConfigMap) Intermediate {
	s.Cfgs = append(s.Cfgs, configMap)
	return s
}

func (s *intermediate) AddSecret(secret corev1.Secret) Intermediate {
	s.Secs = append(s.Secs, secret)
	return s
}

func (s *intermediate) AddEmptyDirVolume(source *corev1.EmptyDirVolumeSource, mountPath string) corev1.VolumeMount {
	if source == nil {
		source = &corev1.EmptyDirVolumeSource{}
	}
	ref := s.NextRef()
	s.AddVolume(corev1.Volume{Name: ref, VolumeSource: corev1.VolumeSource{EmptyDir: source}})
	return corev1.VolumeMount{Name: ref, MountPath: mountPath}
}

// Handling files

func (s *intermediate) AddTextFile(file string, mode *int32) (corev1.VolumeMount, error) {
	mount, _, err := s.Files.AddTextFile(file, mode)
	return mount, err
}

func (s *intermediate) AddBinaryFile(file []byte, mode *int32) (corev1.VolumeMount, error) {
	mount, _, err := s.Files.AddFile(file, mode)
	return mount, err
}
