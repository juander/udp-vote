package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/juander/udp-vote/internal/server"
)

func main() {
	logFile, err := os.OpenFile("logs/server_udp.log", os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666)
	if err != nil {
		log.Fatal("Erro ao abrir log:", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	fmt.Println("=== SERVIDOR UDP ===")
	fmt.Println("Logs: logs/server_udp.log")

	opcoes := []string{"A", "B", "C"}
	srv := server.NewUDPServer(true, opcoes)

	go func() {
		time.Sleep(5 * time.Second)
		fmt.Println("Iniciando votação (300s)...")
		srv.StartVoting(300)
	}()

	srv.Start(":9000")
}
