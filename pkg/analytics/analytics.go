package analytics

import (
	"crypto/md5"
	"encoding/hex"
	"os"

	"github.com/denisbrodbeck/machineid"
	v1 "github.com/mjpitz/go-ga/client/v1"
	"github.com/mjpitz/go-ga/client/v1/gatypes"

	"github.com/kubeshop/testkube/cmd/tools/commands"
)

const testkubeTrackingID = "UA-221444687-1"

func SendAnonymousInfo() {
	client := v1.NewClient(testkubeTrackingID, "golang")
	payload := &gatypes.Payload{
		HitType:                           "event",
		NonInteractionHit:                 true,
		DisableAdvertisingPersonalization: true,
		Users: gatypes.Users{
			ClientID: MachineID(),
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
	client.SendPost(payload)
}

func SendAnonymouscmdInfo() {
	client := v1.NewClient(testkubeTrackingID, "golang")
	payload := &gatypes.Payload{
		HitType:                           "event",
		DataSource:                        "CLI",
		DisableAdvertisingPersonalization: true,
		Users: gatypes.Users{
			ClientID: MachineID(),
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
	client.SendPost(payload)
}

func MachineID() string {
	id, _ := generate()
	return id
}

// Generate returns protected id for the current machine
func generate() (string, error) {
	id, err := machineid.ProtectedID("testkube")
	if err != nil {
		return fromHostname()
	}
	return id, err
}

// fromHostname generates a machine id hash from hostname
func fromHostname() (string, error) {
	name, err := os.Hostname()
	if err != nil {
		return "", err
	}
	sum := md5.Sum([]byte(name))
	return hex.EncodeToString(sum[:]), nil
}
