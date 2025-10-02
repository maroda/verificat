package verificat

import (
	"fmt"

	"github.com/honeycombio/otel-config-go/otelconfig"
)

func InitOTel() (func(), error) {
	otelShutdown, err := otelconfig.ConfigureOpenTelemetry()
	if err != nil {
		return nil, fmt.Errorf("failed to configure OpenTelemetry: %w", err)
	}
	return func() { otelShutdown() }, nil
}
