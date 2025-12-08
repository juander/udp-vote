package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/yLukas077/udp-vote/internal/server"
)

func main() {
	// Redireciona logs para arquivo persistente
	logFile, err := os.OpenFile("logs/server.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		log.Fatal("Erro ao abrir log:", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	fmt.Println("=== SERVIDOR UDP DE VOTAÇÃO ===")
	fmt.Println("Logs: logs/server.log")
	fmt.Println("Modo: Assíncrono (Canal + Worker)")

	// Inicia servidor UDP
	srv := server.NewServer()

	// Inicia votação após 5 segundos com duração de 60 segundos
	go func() {
		time.Sleep(5 * time.Second)
		fmt.Println("Iniciando votação (60 segundos)...")
		srv.StartVoting(60)
	}()

	srv.Start(":9000")
}