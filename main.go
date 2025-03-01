package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	controller "github.com/ArthurMaverick/kube-request/internal"
)

// openBrowser tries to open the specified URL in a browser window.
func openBrowser(url string) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		// Unsupported platform
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Failed to open browser: %v", err)
	}
}

func main() {
	podController := controller.NewPodController()

	mux := http.NewServeMux()
	mux.HandleFunc("/", podController.HandlePods)
	mux.HandleFunc("/change-context", podController.HandleChangeContext)

	// Serve on port 8081
	srv := &http.Server{
		Addr:              ":8081",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Start the server in a goroutine
	go func() {
		log.Printf("Server running on http://localhost%s", srv.Addr)

		// Open the browser as soon as the server starts
		openBrowser("http://localhost" + srv.Addr)

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Error starting server: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-ctx.Done()
	log.Println("Received interrupt signal, initiating graceful shutdown...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Forced shutdown: %v", err)
	}

	log.Println("Server stopped gracefully.")
}
