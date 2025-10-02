package main

import (
	"log/slog"
	"net/http"
	"os"

	vo "github.com/maroda/verificat/obvy"
	verificat "github.com/maroda/verificat/server"
)

const (
	dbFileName = "almanac.db.json"
	app        = "verificat"
	runPort    = "4330"
	llvl       = slog.LevelInfo
)

func init() {
	verificat.CreateLogger(llvl, app)
	slog.Info("Starting Verificat: Production Readiness Verification", slog.String("port", runPort))
}

// Main connects a local JSON database to a running API service.
func main() {
	// Init OpenTelemetry here first
	shutdown, err := vo.InitOTel()
	if err != nil {
		slog.Error("Error initializing OTel", slog.Any("error", err))
	}
	defer shutdown()

	// Open JSON Database file
	db, err := os.OpenFile(dbFileName, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		slog.Error("Error opening database file", slog.Any("error", err), slog.String("filename", dbFileName))
		os.Exit(1)
	}

	// Connect File System Storage operations to JSON Database
	store, err := verificat.NewFSStore(db)
	if err != nil {
		slog.Error("Error initializing FSStore", slog.Any("error", err))
		os.Exit(1)
	}

	// A NewVerificationServ is configured with the database on local disk
	server := verificat.NewVerificationServ(store)
	if err := http.ListenAndServe(":"+runPort, server); err != nil {
		slog.Error("Could not start Verification Service", slog.Any("error", err))
		os.Exit(1)
	}
}
