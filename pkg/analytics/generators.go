package analytics

import (
	"crypto/md5"
	"encoding/hex"
	"os"

	"github.com/denisbrodbeck/machineid"
)

// MachineID returns unique user machine ID
func MachineID() string {
	id, err := machineid.ProtectedID("testkube")
	// fallback to hostname based machine id in case of error
	if err != nil {
		name, err := os.Hostname()
		if err != nil {
			return "default-machine-id"
		}
		sum := md5.Sum([]byte(name))
		return hex.EncodeToString(sum[:])
	}
	return id
}
