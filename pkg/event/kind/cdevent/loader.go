package cdevent

import (
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.uber.org/zap"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/event/kind/common"
	"github.com/kubeshop/testkube/pkg/log"
)

var _ common.ListenerLoader = (*CDEventLoader)(nil)

func NewCDEventLoader(target, clusterID, defaultNamespace, dashboardURI string, events []testkube.EventType) (*CDEventLoader, error) {
	c, err := cloudevents.NewClientHTTP(cloudevents.WithTarget(target))
	if err != nil {
		return nil, err
	}

	return &CDEventLoader{
		Log:              log.DefaultLogger,
		events:           events,
		client:           c,
		clusterID:        clusterID,
		defaultNamespace: defaultNamespace,
		dashboardURI:     dashboardURI,
	}, nil
}

// CDEventLoader is a reconciler for cdevent events for now it returns single listener for cdevent
type CDEventLoader struct {
	Log              *zap.SugaredLogger
	events           []testkube.EventType
	client           cloudevents.Client
	clusterID        string
	defaultNamespace string
	dashboardURI     string
}

func (r *CDEventLoader) Kind() string {
	return "cdevent"
}

// Load returns single listener for cd eventt
func (r *CDEventLoader) Load() (listeners common.Listeners, err error) {
	return common.Listeners{NewCDEventListener("cdevent", "", r.clusterID, r.defaultNamespace, r.dashboardURI, r.events, r.client)}, nil
}
