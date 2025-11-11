package services

import (
	"context"
	"os"
	"time"

	"github.com/kubeshop/testkube/pkg/log"
	configRepo "github.com/kubeshop/testkube/pkg/repository/config"
	"github.com/kubeshop/testkube/pkg/telemetry"
	"github.com/kubeshop/testkube/pkg/utils/text"
	"github.com/kubeshop/testkube/pkg/version"
)

const (
	heartbeatInterval = time.Hour
)

func HandleTelemetryHeartbeat(ctx context.Context, clusterId string, configMapConfig configRepo.Repository) {
	telemetryEnabled, _ := configMapConfig.GetTelemetryEnabled(ctx)
	if telemetryEnabled {
		out, err := telemetry.SendServerStartEvent(clusterId, version.Version)
		if err != nil {
			log.DefaultLogger.Debug("telemetry send error", "error", err.Error())
		} else {
			log.DefaultLogger.Debugw("sending telemetry server start event", "output", out)
		}
	}

	ticker := time.NewTicker(heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			telemetryEnabled, _ = configMapConfig.GetTelemetryEnabled(ctx)
			if telemetryEnabled {
				l := log.DefaultLogger.With("measurmentId", telemetry.TestkubeMeasurementID, "secret", text.Obfuscate(telemetry.TestkubeMeasurementSecret))
				host, err := os.Hostname()
				if err != nil {
					l.Debugw("getting hostname error", "hostname", host, "error", err)
				}
				out, err := telemetry.SendHeartbeatEvent(host, version.Version, clusterId)
				if err != nil {
					l.Debugw("sending heartbeat telemetry event error", "error", err)
				} else {
					l.Debugw("sending heartbeat telemetry event", "output", out)
				}
			}
		}
	}
}
