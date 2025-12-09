package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

///////////////////////////////////////////////////////////////////////////////
// ESTRUTURAS DE DADOS
///////////////////////////////////////////////////////////////////////////////

// Formato JSON trocado com o servidor
type Message struct {
	Type       string         `json:"type"`          // REGISTER, VOTE, ACK, ERROR, BROADCAST...
	ClientID   string         `json:"client_id"`     // nome/ID do cliente
	VoteOption string         `json:"vote,omitempty"`// opção de voto enviada ao servidor
	Message    string         `json:"message,omitempty"` // textos de confirmação/erro do servidor
	VoteCounts map[string]int `json:"vote_counts,omitempty"` // placar recebido no broadcast
	SeqNum     int            `json:"seq_num,omitempty"`     // número sequencial para detecção de perda
}

// Estatísticas locais do cliente (para medir UDP)
type Stats struct {
	m sync.Mutex

	sent       int // total de votos enviados
	confirmed  int // ACK recebidos
	broadcasts int // quantos broadcasts chegaram
	lost       int // pacotes perdidos detectados pelo SeqNum
	lastSeq    int // último número de broadcast recebido
}

// Métodos simples com travamento para segurança concorrente
func (s *Stats) addVote()     { s.m.Lock(); s.sent++; s.m.Unlock() }
func (s *Stats) confirm()     { s.m.Lock(); s.confirmed++; s.m.Unlock() }
func (s *Stats) addBroadcast(){ s.m.Lock(); s.broadcasts++; s.m.Unlock() }

// Detecta perda de pacotes comparando SeqNum com anterior
func (s *Stats) seqCheck(n int) {
	s.m.Lock()

	// Se o número recebido pulou algum, foram perdidos pacotes
	if s.lastSeq > 0 && n > s.lastSeq+1 {
		s.lost += n - s.lastSeq - 1
	}
	s.lastSeq = n

	s.m.Unlock()
}

// Exibe relatório completo
func (s *Stats) Print() {
	s.m.Lock()
	defer s.m.Unlock()

	fmt.Println("\n===== UDP STATS =====")
	fmt.Println("Votos enviados:", s.sent)
	fmt.Println("Confirmados   :", s.confirmed)
	fmt.Println("Não confirm. :", s.sent-s.confirmed)   // diferença = possíveis perdas
	fmt.Println("Broadcasts   :", s.broadcasts)
	fmt.Println("Pacotes perd.:", s.lost)

	total := s.broadcasts + s.lost
	if total > 0 {
		fmt.Printf("Perda estimada: %.2f%%\n", float64(s.lost)/float64(total)*100)
	}
	fmt.Println("=====================\n")
}

///////////////////////////////////////////////////////////////////////////////
// MAIN
///////////////////////////////////////////////////////////////////////////////

func main() {
	// Exige nome como argumento
	if len(os.Args) < 2 {
		fmt.Println("Uso: go run client.go <nome>")
		return
	}
	name := os.Args[1]
	stats := &Stats{}

	// Cria conexão UDP com o servidor
	conn, _ := net.Dial("udp", "localhost:9000")
	defer conn.Close()

	// Thread paralela que escuta mensagens do servidor
	go listen(conn, stats)

	// Primeiro passo = registrar o cliente
	send(conn, "REGISTER", name, "")
	fmt.Println("Conectado. Comandos: VOTE <X> | STATS | QUIT")

	input := bufio.NewScanner(os.Stdin)

	// Loop do terminal do usuário
	for {
		fmt.Print(">> ")
		input.Scan()
		cmd := input.Text()

		// Interpretação dos comandos locais
		switch {
		case cmd == "STATS":     // imprime métricas UDP
			stats.Print()

		case cmd == "QUIT":      // finalizar cliente
			stats.Print()
			return

		case strings.HasPrefix(cmd, "VOTE "):
			option := strings.TrimPrefix(cmd, "VOTE ")
			stats.addVote()                  // soma tentativa
			send(conn, "VOTE", name, option) // envia para o servidor

		default:
			fmt.Println("Comandos: VOTE <A/B/...>, STATS, QUIT")
		}
	}
}

///////////////////////////////////////////////////////////////////////////////
// LISTENER: RECEBE MENSAGENS DO SERVIDOR
///////////////////////////////////////////////////////////////////////////////

func listen(conn net.Conn, stats *Stats) {
	buf := make([]byte, 4096)

	for {
		// Define timeout para não travar para sempre
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))

		n, err := conn.Read(buf) // aguarda dado do servidor
		if err != nil { continue }

		var msg Message
		if json.Unmarshal(buf[:n], &msg) != nil { continue }

		// Processa pelo tipo
		switch msg.Type {

		case "ACK": // confirmação de voto/registro
			stats.confirm()
			fmt.Printf("\n[OK] %s\n>> ", msg.Message)

		case "ERROR": // erro validado no servidor
			fmt.Printf("\n[ERRO] %s\n>> ", msg.Message)

		case "BROADCAST": // placar ao vivo
			stats.addBroadcast()
			stats.seqCheck(msg.SeqNum) // detecta perda de broadcast
			fmt.Printf("\n📡 Parcial #%d %v\n>> ", msg.SeqNum, msg.VoteCounts)
		}
	}
}

///////////////////////////////////////////////////////////////////////////////
// FUNÇÃO PARA ENVIAR MENSAGENS
///////////////////////////////////////////////////////////////////////////////

// send cria um JSON e envia pelo UDP
func send(c net.Conn, t, id, opt string) {
	data,_ := json.Marshal(Message{Type:t, ClientID:id, VoteOption:opt})
	c.Write(data)
}
