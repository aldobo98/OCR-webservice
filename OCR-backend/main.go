package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aldobo98/OCR-backend/internal/database/postgresql"
	"github.com/aldobo98/OCR-backend/internal/job"
	"github.com/aldobo98/OCR-backend/internal/storage/aws_s3"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	database := postgresql.NewPostgres(logger, "", "", "", "", "", "")
	if database == nil {
		logger.Error("Failed to create database")
		return
	}
	db, err := job.NewDatabase(database)
	if err != nil {
		logger.Error("Failed to initialize database", "error", err)
		return
	}
	s3_storage := aws_s3.NewAWS_S3(logger, "", "ocr", "")
	storage, err := job.NewStorage(s3_storage)
	if err != nil {
		logger.Error("Failed to initialize service", "error", err)
		return
	}
	mux := http.NewServeMux()

	//Serve already built frontend
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/", fs)

	mux.Handle("POST /api/jobs", job.CreateJobHandler(logger, storage, db))
	mux.Handle("POST /api/jobs/{jobID}/start", job.CreateOcrStartHandler(logger, storage, db)) //Elkezdjük a feldolgozást
	// "POST /internal/jobs/callback" -> itt fogjuk visszakapni az OCR felismerés eredményét

	//Start HTTP server in a goroutine
	s := http.Server{Addr: ":8080", Handler: mux}
	go func() {
		logger.Info("Server listening on http://:8080")
		err := s.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			logger.Error("HTTP server error", "error", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutdown signal received")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	err = s.Shutdown(ctx)
	if err != nil {
		logger.Info("Graceful server shutdown failed with", "error", err)
	} else {
		logger.Info("Graceful server sutdown succeeded")
	}
}
