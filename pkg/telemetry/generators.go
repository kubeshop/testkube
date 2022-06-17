package telemetry

import (
	"crypto/md5"
	"encoding/hex"
	"os"

	"github.com/denisbrodbeck/machineid"
	"github.com/kubeshop/testkube/pkg/log"
)

// GetMachineID returns unique user machine ID
func GetMachineID() string {
	id, err := machineid.ProtectedID("testkube")
	// fallback to hostname based machine id in case of error
	if err != nil {
		log.DefaultLogger.Debugw("error while generating machines protected id", "error", err)
		name, err := os.Hostname()
		if err != nil {
			return "default-machine-id"
		}
		sum := md5.Sum([]byte(name))
		return hex.EncodeToString(sum[:])
	}
	return id
}
