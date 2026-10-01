package testworkflowprocessor

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube/api/testworkflows/v1"
	"github.com/kubeshop/testkube/pkg/executioncache/volume"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowconfig"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/stage"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowresolver"
)

//go:generate go tool mockgen -destination=./mock_intermediate.go -package=testworkflowprocessor "github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor" Intermediate
type Intermediate interface {
	RefCounter

	ContainerDefaults() stage.Container
	PodConfig() testworkflowsv1.PodConfig
	JobConfig() testworkflowsv1.JobConfig

	ConfigMaps() []corev1.ConfigMap
	Secrets() []corev1.Secret
	Volumes() []corev1.Volume
	Pvcs() map[string]corev1.PersistentVolumeClaim

	AppendJobConfig(cfg *testworkflowsv1.JobConfig) Intermediate
	AppendPodConfig(cfg *testworkflowsv1.PodConfig) Intermediate
	AppendPvcs(cfg map[string]corev1.PersistentVolumeClaimSpec) Intermediate
	AppendStepCacheVolume(cfg *testworkflowconfig.StepCacheVolumeConfig, resourceID string) Intermediate

	AddConfigMap(configMap corev1.ConfigMap) Intermediate
	AddSecret(secret corev1.Secret) Intermediate
	AddVolume(volume corev1.Volume) Intermediate

	AddEmptyDirVolume(source *corev1.EmptyDirVolumeSource, mountPath string) corev1.VolumeMount

	// StepCacheVolumeMount returns a mount of the operator's shared step-cache volume,
	// and whether there is one to mount.
	//
	// The volume is added to the pod on the first call and reused afterwards: a
	// workflow with several cached steps needs one volume and several mounts of it,
	// where AddEmptyDirVolume's one-volume-per-mount shape would attach the same claim
	// once per step.
	//
	// readOnly and subPath are the only confinement there is, and they are enforced by
	// kubelet rather than by the toolkit choosing to behave. Both cache stages are
	// pure, so action.Group merges them into the step's own container and the step's
	// own command ends up holding whatever is mounted here.
	StepCacheVolumeMount(mountPath, subPath string, readOnly bool) (corev1.VolumeMount, bool)

	// StepCacheInboxName is what this execution's inbox is called from the volume root,
	// which a pointer needs because the write mount is a subPath and so hides it.
	StepCacheInboxName() string

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
	Secs []corev1.Secret                         `expr:"force"`
	Cfgs []corev1.ConfigMap                      `expr:"force"`
	Ps   map[string]corev1.PersistentVolumeClaim `expr:"force"`

	// Storing files
	Files ConfigMapFiles `expr:"include"`

	// Default sizeLimit for emptyDir volumes
	DefaultEmptyDirSizeLimit *resource.Quantity

	// StepCacheVolume is the operator's shared cache volume, or nil when step
	// dependency caches go to the object store whole.
	StepCacheVolume *testworkflowconfig.StepCacheVolumeConfig
	// stepCacheVolumeName is the pod volume once it has been added, so that several
	// cached steps share one rather than attaching the claim once each.
	stepCacheVolumeName string
	// stepCacheInboxName is what this execution's inbox is called from the volume root.
	stepCacheInboxName string
}

func NewIntermediate(defaultEmptyDirSizeLimit string) Intermediate {
	ref := NewRefCounter()
	var defaultLimit *resource.Quantity
	if defaultEmptyDirSizeLimit != "" {
		// Bundle validates worker config up front; this extra parse keeps direct
		// NewIntermediate callers tolerant of invalid input (invalid values are ignored).
		if parsedLimit, err := resource.ParseQuantity(defaultEmptyDirSizeLimit); err == nil {
			defaultLimit = &parsedLimit
		}
	}
	return &intermediate{
		RefCounter:               ref,
		Root:                     stage.NewGroupStage("", true),
		Container:                stage.NewContainer(),
		Files:                    NewConfigMapFiles(fmt.Sprintf("{{resource.id}}-%s", ref.NextRef()), nil),
		Ps:                       make(map[string]corev1.PersistentVolumeClaim),
		DefaultEmptyDirSizeLimit: defaultLimit,
	}
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
	volumes := make([]corev1.Volume, 0, len(s.Pod.Volumes)+len(s.Files.Volumes()))
	for i := range s.Pod.Volumes {
		volume := *s.Pod.Volumes[i].DeepCopy()
		if volume.EmptyDir != nil {
			s.applyDefaultEmptyDirSizeLimit(volume.EmptyDir)
		}
		volumes = append(volumes, volume)
	}
	return append(volumes, s.Files.Volumes()...)
}

func (s *intermediate) Pvcs() map[string]corev1.PersistentVolumeClaim {
	return s.Ps
}

func (s *intermediate) AppendJobConfig(cfg *testworkflowsv1.JobConfig) Intermediate {
	s.Job = *testworkflowresolver.MergeJobConfig(&s.Job, cfg)
	return s
}

func (s *intermediate) AppendPodConfig(cfg *testworkflowsv1.PodConfig) Intermediate {
	s.Pod = *testworkflowresolver.MergePodConfig(&s.Pod, cfg)
	return s
}

func (s *intermediate) AppendPvcs(cfg map[string]corev1.PersistentVolumeClaimSpec) Intermediate {
	for name, spec := range cfg {
		s.Ps[name] = corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("{{resource.root}}-%s", s.NextRef()),
			},
			Spec: spec,
		}
	}
	return s
}

func (s *intermediate) AppendStepCacheVolume(cfg *testworkflowconfig.StepCacheVolumeConfig, resourceID string) Intermediate {
	// Both are needed: without a claim there is no volume, and without a resource id a
	// committed entry could not be named from the volume root, so no reader would ever
	// find it.
	if cfg != nil && cfg.ClaimName != "" && resourceID != "" {
		s.StepCacheVolume = cfg
		s.stepCacheInboxName = volume.InboxFor(resourceID)
	}
	return s
}

// StepCacheInboxName is what this execution's inbox is called from the volume root.
func (s *intermediate) StepCacheInboxName() string {
	return s.stepCacheInboxName
}

func (s *intermediate) StepCacheVolumeMount(mountPath, subPath string, readOnly bool) (corev1.VolumeMount, bool) {
	if s.StepCacheVolume == nil {
		return corev1.VolumeMount{}, false
	}
	if s.stepCacheVolumeName == "" {
		s.stepCacheVolumeName = s.NextRef()
		s.AddVolume(corev1.Volume{
			Name: s.stepCacheVolumeName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: s.StepCacheVolume.ClaimName,
				},
			},
		})
	}
	return corev1.VolumeMount{
		Name:      s.stepCacheVolumeName,
		MountPath: mountPath,
		SubPath:   subPath,
		ReadOnly:  readOnly,
	}, true
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

func (s *intermediate) applyDefaultEmptyDirSizeLimit(source *corev1.EmptyDirVolumeSource) {
	if source == nil || source.SizeLimit != nil || s.DefaultEmptyDirSizeLimit == nil {
		return
	}
	qty := *s.DefaultEmptyDirSizeLimit
	source.SizeLimit = &qty
}

func (s *intermediate) AddEmptyDirVolume(source *corev1.EmptyDirVolumeSource, mountPath string) corev1.VolumeMount {
	if source == nil {
		source = &corev1.EmptyDirVolumeSource{}
	}
	s.applyDefaultEmptyDirSizeLimit(source)
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
