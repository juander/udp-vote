package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"time"
)

func main() {
	conn, err := net.Dial("udp", "localhost:9000")
	if err != nil {
		fmt.Println("Erro ao conectar:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Conectado ao servidor UDP!")

	// Goroutine dedicada para leitura assíncrona
	go func() {
		for {
			buffer := make([]byte, 1024)
			n, _, err := conn.ReadFrom(buffer)
			if err != nil {
				fmt.Println("Erro ao ler do servidor:", err)
				return
			}
			fmt.Println("\n[SERVIDOR]:", string(buffer[:n]))
			fmt.Print(">> ")
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)

	// Handshake: envia NOME do cliente
	fmt.Print("Digite seu NOME para entrar: ")
	scanner.Scan()
	id := scanner.Text()

	// Envia o nome do cliente
	fmt.Fprintf(conn, "%s\n", id)

	// Loop de envio de votos
	for {
		fmt.Print("Digite seu VOTO (A, B, C) ou 'sair' para encerrar: ")
		if !scanner.Scan() {
			break
		}
		text := scanner.Text()
		if text == "sair" {
			break
		}
		fmt.Fprintf(conn, "%s\n", text)
		time.Sleep(100 * time.Millisecond) // Simula um atraso entre os votos
	}

	fmt.Println("Encerrando cliente.")
}