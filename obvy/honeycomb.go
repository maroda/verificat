package verificat

import (
	"fmt"
	"log/slog"

	"github.com/honeycombio/otel-config-go/otelconfig"
)

func InitOTel() (func(), error) {
	otelShutdown, err := otelconfig.ConfigureOpenTelemetry()
	config := otelconfig.DefaultExporterEndpoint
	slog.Info("Initializing otel", slog.String("endpoint", config))
	if err != nil {
		return nil, fmt.Errorf("failed to configure OpenTelemetry: %w", err)
	}
	return func() { otelShutdown() }, nil
}
