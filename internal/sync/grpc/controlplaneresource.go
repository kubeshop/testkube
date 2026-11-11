package grpc

import (
	"context"
	"fmt"

	"google.golang.org/grpc/metadata"

	"github.com/kubeshop/testkube/pkg/controlplane/declarative"
	syncv1 "github.com/kubeshop/testkube/pkg/proto/testkube/sync/v1"
)

// UpdateOrCreateControlPlaneResource sends a Control Plane resource, such as an Insights board, to
// the Control Plane. The object goes as declarative.ApplyPayload makes it for a PUT to the
// declarative endpoints, so the Control Plane reads it with the same rules. The caller normalizes it
// first.
func (c Client) UpdateOrCreateControlPlaneResource(ctx context.Context, obj declarative.Object) error {
	kind, err := declarative.KindOf(obj)
	if err != nil {
		return err
	}
	payload, err := declarative.ApplyPayload(obj)
	if err != nil {
		return fmt.Errorf("encode %s %q: %w", kind.Kind, obj.GetName(), err)
	}
	resource, name := kind.Resource, obj.GetName()

	// Execute with our own call timeout context to prevent stalling out.
	callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
	defer cancel()
	callCtx = metadata.AppendToOutgoingContext(callCtx, "organization-id", c.OrganizationId)

	if _, err := c.client.UpdateOrCreate(callCtx, &syncv1.UpdateOrCreateRequest{
		Payload: &syncv1.UpdateOrCreateRequest_ControlPlaneResource{
			ControlPlaneResource: &syncv1.ControlPlaneResource{
				Resource: &resource,
				Name:     &name,
				Payload:  payload,
			},
		},
	}, c.callOpts...); err != nil {
		return fmt.Errorf("send request to update or create %s: %w", kind.Kind, translateError(err))
	}
	return nil
}

// DeleteControlPlaneResource tells the Control Plane that a Control Plane resource has been removed
// from the observable scope of this Agent.
func (c Client) DeleteControlPlaneResource(ctx context.Context, kind declarative.Kind, name string) error {
	resource := kind.Resource

	callCtx, cancel := context.WithTimeout(ctx, c.callTimeout)
	defer cancel()
	callCtx = metadata.AppendToOutgoingContext(callCtx, "organization-id", c.OrganizationId)

	if _, err := c.client.Delete(callCtx, &syncv1.DeleteRequest{
		Id: &syncv1.DeleteRequest_ControlPlaneResource{
			ControlPlaneResource: &syncv1.ControlPlaneResourceId{
				Resource: &resource,
				Name:     &name,
			},
		},
	}, c.callOpts...); err != nil {
		return fmt.Errorf("send request to delete %s: %w", kind.Kind, translateError(err))
	}
	return nil
}
