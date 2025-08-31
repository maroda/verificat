package main

import (
	"log"
	"log/slog"
	"net/http"
	"os"

	verificat "github.com/maroda/verificat/server"
)

const (
	dbFileName = "almanac.db.json"
	app        = "verificat"
	llvl       = slog.LevelDebug // TODO: this should be configurable
	runPort    = "4330"          // TODO: this should be configurable
)

func init() {
	verificat.CreateLogger(llvl, app)
	slog.Info("Starting Verificat: Production Readiness Verification", slog.String("port", runPort))
}

// Main connects a local JSON database to a running API service.
func main() {
	// Open JSON Database file
	db, err := os.OpenFile(dbFileName, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Fatalf("problem opening %s %v", dbFileName, err)
	}

	// Connect File System Storage operations to JSON Database
	store, err := verificat.NewFSStore(db)
	if err != nil {
		log.Fatalf("problem creating file system service store, %v ", err)
	}

	// A NewVerificationServ is configured with the database on local disk
	server := verificat.NewVerificationServ(store)
	if err := http.ListenAndServe(":"+runPort, server); err != nil {
		slog.Error("Server Crash")
	}
}
