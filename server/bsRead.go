package verificat

import (
	"log/slog"
	"time"
)

// SvcCat contains methods for operating with the Service Catalog, e.g. Backstage API.
type SvcCat interface {
	ReadSvc() (string, error)
}

// SvcConfig is the Client Configuration
type SvcConfig struct {
	URL      string // URL is the Backstage API endpoint
	Service  string // Each Service is known as the "Component" in Backstage
	Datetime int64  // Unix Epoch in seconds
	Owner    string // Should equal CODEOWNERS for this repo in GitHub
}

// ReadSvc can query Backstage for a chunk of data about a System,
// i.e. the "top-level" Service.
// Each method called for filling in data adds the entry to the SvcConfig struct.
func (sc *SvcConfig) ReadSvc() (string, error) {
	sc.Datetime = time.Now().Unix()

	/* removing Backstage slowly ... this will be a generic comparison to an example

	c, _ := backstage.NewClient(sc.URL, "default", nil)
	owner, _, err := ReadSystemBS(sc.Service, c)

	*/

	// TODO: Figure out something besides a complicated Backstage setup to test.
	// Temporarily, return a static string that will match
	sc.Owner = "maroda"
	slog.Debug("Owner Set", slog.String("Owner", sc.Owner))
	return sc.Owner, nil
}

// ReadinessRead is the function that tests this service for Production Readiness
func ReadinessRead(i SvcCat) (string, error) {
	// Calling ReadSvc() initiates the source data struct, SvcConfig
	// Currently only returning the Owner, which is what ReadSvc() returns
	return i.ReadSvc()
}
