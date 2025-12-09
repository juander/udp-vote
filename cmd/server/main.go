package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/juander/udp-vote/internal/server"
)

func main() {
	// Logs em arquivo
	logFile, err := os.OpenFile("logs/server_udp.log",
		os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		log.Fatal("Erro ao abrir arquivo de log:", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	fmt.Println("=== SERVIDOR UDP DE VOTAÇÃO ===")
	fmt.Println("Logs salvos em: logs/server_udp.log\n")

	// Opções da votação
	opcoes := []string{"A", "B", "C"}

	// Cria servidor sempre assíncrono
	srv := server.NewUDPServer(opcoes)

	// Inicia votação automaticamente após 5 segundos
	go func() {
		time.Sleep(5 * time.Second)
		fmt.Println("Iniciando votação (300s)...")
		srv.StartVoting(300)
	}()

	// Escuta na porta UDP
	srv.Start(":9000")
}
