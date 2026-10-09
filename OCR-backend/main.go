package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aldobo98/OCR-backend/handler"
	"github.com/aldobo98/ocr-common/aws_s3"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	s3_storage := aws_s3.NewAWS_S3(logger, "http://10.8.0.1:31374", "ocr", "")
	err := s3_storage.Ensurebucket()
	if err != nil {
		logger.Error("Failed to ensure s3 bucket", "error", err)
	}

	//Serve already built frontend
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)

	//A presigned URL requestet a /api/uploadurl endpoint fogja kiszolgálni
	http.HandleFunc("/api/uploadurl", handler.CreateJobRequestHandler(logger, s3_storage))
	//Start HTTP server in a goroutine
	s := http.Server{Addr: ":8080"}
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
