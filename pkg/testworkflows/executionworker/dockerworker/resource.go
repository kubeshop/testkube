package dockerworker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	dockerclient "github.com/docker/docker/client"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	errors2 "github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	"github.com/kubeshop/testkube/internal/common"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor"
	"github.com/kubeshop/testkube/pkg/testworkflows/testworkflowprocessor/constants"
)

type dockerContainerInput struct {
	config  container.Config
	host    container.HostConfig
	network network.NetworkingConfig
	name    string
}

type dockerContainer struct {
	id   string
	name string
}

type dockerDeployment struct {
	volumes    []volume.Volume
	containers []dockerContainer
}

// TODO: Handle terminationLog?
func Deploy(ctx context.Context, client *dockerclient.Client, bundle *testworkflowprocessor.Bundle) (*dockerDeployment, error) {
	// Determine IDs
	resourceId := bundle.Job.Spec.Template.Labels[constants.ResourceIdLabelName]
	rootResourceId := bundle.Job.Spec.Template.Labels[constants.RootResourceIdLabelName]

	// Determine volumes
	volumes := make([]volume.CreateOptions, 0, len(bundle.Job.Spec.Template.Spec.Volumes))
	for _, v := range bundle.Job.Spec.Template.Spec.Volumes {
		if v.EmptyDir == nil {
			return nil, errors.New("only emptyDir volumes are supported yet")
		}
		volumes = append(volumes, volume.CreateOptions{
			Labels: map[string]string{
				constants.ResourceIdLabelName:     resourceId,
				constants.RootResourceIdLabelName: rootResourceId,
			},
			Driver: "local",
			Name:   fmt.Sprintf("%s-%s", resourceId, v.Name),
		})
	}

	// List all the expected containers
	k8sContainers := append(bundle.Job.Spec.Template.Spec.InitContainers, bundle.Job.Spec.Template.Spec.Containers...)

	// Determine containers
	containers := make([]dockerContainerInput, 0, len(k8sContainers))
	for _, cn := range k8sContainers {
		// Determine user
		// TODO: FSGroup like: add another start 'root' container that will set proper permissions for the volumes
		//user := ""
		//if cn.SecurityContext.RunAsUser != nil {
		//	if cn.SecurityContext.RunAsGroup != nil {
		//		user = fmt.Sprintf("%d:%d", *cn.SecurityContext.RunAsUser, *cn.SecurityContext.RunAsGroup)
		//	} else {
		//		user = fmt.Sprintf("%d", *cn.SecurityContext.RunAsUser)
		//	}
		//}
		user := "root"

		// Prepare the volume mounts
		mounts := common.MapSlice(cn.VolumeMounts, func(vm corev1.VolumeMount) (m mount.Mount) {
			m = mount.Mount{
				Type:     mount.TypeBind,
				Source:   vm.MountPath,
				Target:   vm.MountPath,
				ReadOnly: vm.ReadOnly,
			}
			if vm.SubPath != "" {
				m.VolumeOptions = &mount.VolumeOptions{
					Subpath: vm.SubPath,
				}
			}
			return
		})

		// Prepare the environment variables
		envs := make([]string, 0)
		for _, e := range cn.Env {
			// Resource Metrics Variables
			if strings.HasPrefix(e.Name, "TKI_R_") {
				continue
			}

			// Plain-Text Variables
			if e.ValueFrom == nil {
				envs = append(envs, fmt.Sprintf("%s=%s", e.Name, e.Value))
				continue
			}

			// Resource Metrics
			if e.ValueFrom.ResourceFieldRef != nil {
				switch e.ValueFrom.ResourceFieldRef.Resource {
				case "requests.cpu":
					if cpu := cn.Resources.Requests.Cpu(); cpu != nil && cpu.String() != "" {
						envs = append(envs, fmt.Sprintf("%s=%s", e.Name, cpu.String()))
					}
				case "limits.cpu":
					if cpu := cn.Resources.Limits.Cpu(); cpu != nil && cpu.String() != "" {
						envs = append(envs, fmt.Sprintf("%s=%s", e.Name, cpu.String()))
					}
				case "requests.memory":
					if mem := cn.Resources.Requests.Memory(); mem != nil && mem.String() != "" {
						envs = append(envs, fmt.Sprintf("%s=%s", e.Name, mem.String()))
					}
				case "limits.memory":
					if mem := cn.Resources.Limits.Memory(); mem != nil && mem.String() != "" {
						envs = append(envs, fmt.Sprintf("%s=%s", e.Name, mem.String()))
					}
				default:
					return nil, fmt.Errorf("unsupported resource field reference: %s", e.ValueFrom.ResourceFieldRef.Resource)
				}
			}

			// Internals
			if e.ValueFrom.FieldRef != nil {
				switch e.ValueFrom.FieldRef.FieldPath {
				case "spec.nodeName", "metadata.namespace", "spec.serviceAccountName":
					envs = append(envs, fmt.Sprintf("%s=", e.Name))
				case "metadata.name":
					envs = append(envs, fmt.Sprintf("%s=%s", e.Name, resourceId))
				case constants.InternalAnnotationFieldPath:
					envs = append(envs, fmt.Sprintf("%s=%s", e.Name, bundle.Job.Spec.Template.Annotations[constants.InternalAnnotationName]))
				case constants.SpecAnnotationFieldPath:
					envs = append(envs, fmt.Sprintf("%s=%s", e.Name, bundle.Job.Spec.Template.Annotations[constants.SpecAnnotationName]))
				case constants.SignatureAnnotationFieldPath:
					envs = append(envs, fmt.Sprintf("%s=%s", e.Name, bundle.Job.Spec.Template.Annotations[constants.SignatureAnnotationName]))
				default:
					return nil, fmt.Errorf("unsupported field reference: %s", e.ValueFrom.FieldRef.FieldPath)
				}
				// Ignore the rest
				continue
			}
		}

		// Pull the image if necessary
		shouldPullImage := cn.ImagePullPolicy == "Never"
		if cn.ImagePullPolicy == "IfNotExists" {
			// TODO: find cheaper method?
			_, _, err := client.ImageInspectWithRaw(ctx, cn.Image)
			shouldPullImage = err != nil
		}
		if shouldPullImage {
			pullReader, err := client.ImagePull(ctx, cn.Image, image.PullOptions{})
			if err != nil {
				return nil, errors2.Wrapf(err, "failed to pull image: %s", cn.Image)
			}
			_, _ = io.Copy(os.Stdout, pullReader)
		}

		containers = append(containers, dockerContainerInput{
			name: cn.Name,
			config: container.Config{
				User:       user,
				Image:      cn.Image,
				WorkingDir: cn.WorkingDir,
				Entrypoint: cn.Command,
				Cmd:        cn.Args,
				Env:        envs,
				Labels:     bundle.Job.Spec.Template.Labels,
			},
			host: container.HostConfig{
				Mounts: mounts,
			},
		})
	}

	// Store deployment
	deployment := dockerDeployment{}

	// Clean up all the resources on failure
	rollback := func() {
		for _, v := range deployment.volumes {
			_ = client.VolumeRemove(ctx, v.Name, true)
		}
		for _, c := range deployment.containers {
			_ = client.ContainerRemove(ctx, c.id, container.RemoveOptions{Force: true})
		}
	}

	// Deploy
	// TODO: parallelize
	for _, v := range volumes {
		vv, err := client.VolumeCreate(ctx, v)
		if err != nil {
			rollback()
			return nil, errors2.Wrapf(err, "failed to create volume: %s", v.Name)
		}
		deployment.volumes = append(deployment.volumes, vv)
	}
	for _, c := range containers {
		cc, err := client.ContainerCreate(ctx, &c.config, &c.host, &c.network, &v1.Platform{}, c.name)
		if err != nil {
			rollback()
			return nil, errors2.Wrapf(err, "failed to create container: %s", c.name)
		}
		deployment.containers = append(deployment.containers, dockerContainer{
			id:   cc.ID,
			name: c.name,
		})
	}

	return &deployment, nil
}

//func Orchestrate(ctx context.Context, client *dockerclient.Client, deployment *dockerDeployment) error {
//
//}
//
//func StartContainer(ctx context.Context, client *dockerclient.Client, cn dockerContainer) error {
//	err := client.ContainerStart(ctx, cn.id, container.StartOptions{})
//	if err != nil {
//		return errors2.Wrapf(err, "failed to start container: %s", cn.name)
//	}
//}
