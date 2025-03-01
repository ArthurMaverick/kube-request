package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	controller "github.com/ArthurMaverick/kube-request/internal"
)

func main() {
	podController := controller.NewPodController()

	mux := http.NewServeMux()
	mux.HandleFunc("/", podController.HandlePods)
	mux.HandleFunc("/change-context", podController.HandleChangeContext)

	// Configura o servidor com boas práticas de timeout
	srv := &http.Server{
		Addr:              ":8086",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,  // Tempo máximo para ler headers
		WriteTimeout:      15 * time.Second, // Tempo máximo para escrever resposta
		IdleTimeout:       60 * time.Second, // Tempo máximo para conexões inativas
	}

	// Cria um contexto que será cancelado ao receber SIGINT ou SIGTERM
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Inicia o servidor em uma goroutine
	go func() {
		log.Printf("Server running on http://localhost%s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Erro ao iniciar servidor: %v", err)
		}
	}()

	// Aguarda o sinal de interrupção para iniciar o graceful shutdown
	<-ctx.Done()
	log.Println("Recebido sinal de interrupção, iniciando shutdown gracioso...")

	// Cria um contexto adicional para encerrar o servidor com timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Shutdown forçado: %v", err)
	}

	log.Println("Servidor finalizado com sucesso.")
}
