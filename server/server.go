package verificat

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	vo "github.com/maroda/verificat/obvy"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

const (
	htmlContentType = "text/html"
	jsonContentType = "application/json"
	htmlTemplates   = "templates/*gohtml"
	targetDocTmpl   = "almanac.gohtml"
)

// WMService defines the service and its final checklist score
// The final score is taken from 100.
// If the checklist comes back as true,
// then the score is 0, and this value remains 100.
// If the checklist comes back as false,
// then the score is 1, and this value becomes 99.
type WMService struct {
	Name   string // Service Name
	LastID int    // The last test ID
	Score  int    // The current score (100 - score)
}

type ServiceStore interface {
	GetTriggerID(name string) int     // Retrieve the count of tests done
	TriggerID(name string, score int) // The current run ID, its score
	GetAlmanac() Almanac              // A collection of all services and their scores
}

// VerificationServ is the main brain,
// serving up http and collecting stats
// connected to a specific data store
type VerificationServ struct {
	stats  *vo.StatsInternal // Prometheus metrics
	store  ServiceStore      // The Almanac service database
	tracer trace.Tracer      // otel tracer
	http.Handler
}

// NewVerificationServ initiates the HTTP service and internal stats with prometheus
func NewVerificationServ(store ServiceStore) *VerificationServ {
	v := new(VerificationServ)
	v.store = store
	v.stats = vo.NewStatsInternal()
	v.tracer = otel.Tracer("verification-serv")

	// This will be assigned to the http.Handler in PlayerServer
	// so that the routing is done once at the start, not on every request.
	router := http.NewServeMux()

	// Set up each server endpoint and its associated handler function
	router.Handle("/metrics", v.stats.Handler())
	router.Handle("/almanac", http.HandlerFunc(v.almanacHandler))
	router.Handle("/healthz", http.HandlerFunc(v.healthzHandler))
	router.Handle("/v0/almanac", http.HandlerFunc(v.almanacHandler))
	router.Handle("/v0/", http.HandlerFunc(v.servicesHandler))
	router.Handle("/", http.HandlerFunc(v.homeHandler))

	v.Handler = router

	return v
}

// Healthz handler (/healthz)
// Very simple endpoint for use with readiness and liveness probes
// If the app isn't answering 'ok' at this endpoint, all probes fail.
// TODO: Include a 'liveness' probe that successfully connects to the database
func (v *VerificationServ) healthzHandler(w http.ResponseWriter, r *http.Request) {
	// No /w.WriteHeader(http.StatusAccepted)/ here
	// 200 is the default for w.Write
	w.Header().Set("content-type", htmlContentType)
	w.Write([]byte(`ok`))

	// Prometheus
	methodString := r.Method + ":" + r.RequestURI
	v.stats.RecWWW("200", methodString)
}

// UI homepage handler
// Render the current full Almanac to the home page
func (v *VerificationServ) homeHandler(w http.ResponseWriter, r *http.Request) {
	// OpenTelemetry
	ctx := r.Context()
	user := os.Getuid()
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.Int("user.id", user))
	ctx, span = v.tracer.Start(ctx, "homeHandler")
	defer span.End()

	// Prometheus
	methodString := r.Method + ":" + r.RequestURI
	v.stats.RecWWW("200", methodString)

	// Write response
	w.Header().Set("content-type", htmlContentType)

	// Configure draw output margins and offsets
	sc := &SVGCfg{Gutter: 3, TxtOff: 8, Spacer: 14}

	// Create a full dataset to work with
	// This is where BuildSVG needs to operate first
	currAlmanac := v.store.GetAlmanac()
	aWeb := &AlmanacWeb{
		Title:     "Verificat | Production Readiness Scores",
		Content:   BuildSVG(&currAlmanac, sc),
		FullScore: currAlmanac,
	}

	if err := RenderWeb(w, aWeb, htmlTemplates, targetDocTmpl); err != nil {
		slog.Error("Page could not be rendered", slog.Any("Error", err))
	}

	slog.Info("Homepage",
		slog.String("Method", r.Method),
		slog.String("Path", r.URL.Path),
		slog.Int64("ContentLength", r.ContentLength),
		slog.String("Remote", r.RemoteAddr),
	)
}

// Fetch full almanac handler
// Return the full JSON almanac of WMServices and their verification scores.
func (v *VerificationServ) almanacHandler(w http.ResponseWriter, r *http.Request) {
	// OpenTelemetry
	ctx := r.Context()
	user := os.Getuid()
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.Int("user.id", user))
	ctx, span = v.tracer.Start(ctx, "almanacHandler")
	defer span.End()

	// Write response
	w.Header().Set("content-type", jsonContentType)
	json.NewEncoder(w).Encode(v.store.GetAlmanac())

	// Prometheus
	methodString := r.Method + ":" + r.RequestURI
	v.stats.RecWWW("200", methodString)

	slog.Info("Almanac API",
		slog.String("Method", r.Method),
		slog.String("Path", r.URL.Path),
		slog.Int64("ContentLength", r.ContentLength),
		slog.String("Remote", r.RemoteAddr),
	)
}

// API for service tests handler
// Version 0 (/v0/<SERVICE>)
func (v *VerificationServ) servicesHandler(w http.ResponseWriter, r *http.Request) {
	// OpenTelemetry
	ctx := r.Context()
	user := os.Getuid()
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.Int("user.id", user))
	ctx, span = v.tracer.Start(ctx, "servicesHandler")
	defer span.End()

	// extract this once here, then it's not necessary to pass http.Request
	service := strings.TrimPrefix(r.URL.Path, "/v0/")

	// Based on the method of the HTTP action, do different things.
	// These methods on VerificationServ can pass the handler interfaces around
	switch r.Method {
	case http.MethodPost:
		// Kick off the test and display the results
		v.runVerification(w, service)
	case http.MethodGet:
		// Get last session ID from the database.
		v.showLastID(w, service)
	}

	// Prometheus
	methodString := r.Method + ":" + r.RequestURI
	v.stats.RecWWW("200", methodString)

	slog.Info("Services API",
		slog.String("Method", r.Method),
		slog.String("Path", r.URL.Path),
		slog.Int64("ContentLength", r.ContentLength),
		slog.String("Remote", r.RemoteAddr),
	)
}

// showLastID will display the ID of the most recent verification run for this service.
func (v *VerificationServ) showLastID(w http.ResponseWriter, service string) {
	// GetTriggerID is a method available through the interface
	lastID := v.store.GetTriggerID(service)

	if lastID == 0 {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "No record found for %+v, try using 'POST -X' with the same URL to create one.\n", service)
	}

	fmt.Fprintf(w, "LastID for "+service+": %d\n", lastID)
}

// runVerification. Takes a passed configuration and launches testing.
func (v *VerificationServ) runVerification(w http.ResponseWriter, service string) {
	start := time.Now()
	w.WriteHeader(http.StatusAccepted)

	var err error
	envVar := "BACKSTAGE"
	url := fillEnvVar(envVar)

	// if there's no EnvVar, log an error and go no further
	if url == "ENOENT" {
		slog.Error("Environment Variable not set", slog.String("Key", envVar), slog.String("Value", url))
		return
	}

	// Create a data object for the configuration.
	svcconf := &SvcConfig{URL: url, Service: service}

	// Read the SVC and get the "owner" string back
	// We don't need a return, it updates the struct
	/*

		ReadinessRead currently circumnavigates the backstage code and returns a static value.

	*/
	_, err = ReadinessRead(svcconf)
	if err != nil {
		slog.Error("ReadinessRead Failed", slog.Any("Error", err))
	} else {
		// ReadinessDisplay expects an interface with this struct
		// These values have been filled in by ReadinessRead() above
		// Score is initialized to 100 each time,
		//	then decremented on each failed test
		//	that is handled by ReadinessDisplay.
		stests := &SvcTestDB{Datetime: svcconf.Datetime, Owner: svcconf.Owner, Score: 100}

		// Send test metadata to ReadinessDisplay, which launches tests and displays the results.
		// w == http.ResponseWriter, which satisfies io.Writer
		err = ReadinessDisplay(stests, service, w)
		if err != nil {
			slog.Error("ReadinessDisplay Failed", slog.Any("Error", err))
		}

		// Initiate the TriggerID sequence that is used to set WMService.Score in the database.
		v.store.TriggerID(service, stests.Score)
	}

	elapsed := time.Since(start).Seconds()
	methodString := "Readiness Read: " + service
	v.stats.RecWWW("200", methodString)
	v.stats.PollTimer.Observe(elapsed)
}
