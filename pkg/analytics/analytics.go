package analytics

import (
	"os"

	v1 "github.com/mjpitz/go-ga/client/v1"
	"github.com/mjpitz/go-ga/client/v1/gatypes"

	"github.com/kubeshop/testkube/cmd/tools/commands"
	"github.com/kubeshop/testkube/pkg/telemetry"
)

const testkubeTrackingID = "UA-221444687-1"

func SendAnonymousInfo() {
	client := v1.NewClient(testkubeTrackingID, "golang")
	ping := &gatypes.Payload{
		HitType:                           "event",
		NonInteractionHit:                 true,
		DisableAdvertisingPersonalization: true,
		Users: gatypes.Users{
			ClientID: telemetry.MachineID(),
		},
		Event: gatypes.Event{
			EventCategory: "beacon",
			EventAction:   "testkube-heartbeat",
		},
		Apps: gatypes.Apps{
			ApplicationName:    "testkube",
			ApplicationVersion: commands.Version,
		},
	}
	client.SendPost(ping)
}

func SendAnonymouscmdInfo() {
	client := v1.NewClient(testkubeTrackingID, "golang")
	ping := &gatypes.Payload{
		HitType:                           "event",
		DataSource:                        "CLI",
		DisableAdvertisingPersonalization: true,
		Users: gatypes.Users{
			ClientID: telemetry.MachineID(),
		},
		Event: gatypes.Event{
			EventCategory: "command",
			EventAction:   "execution",
		},
		Apps: gatypes.Apps{
			ApplicationName:    "testkube",
			ApplicationVersion: commands.Version,
		},
		CustomDimensions: gatypes.StringList(os.Args),
	}
	client.SendPost(ping)
}
