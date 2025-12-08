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

type Message struct {
	Type       string         `json:"type"`
	ClientID   string         `json:"client_id"`
	VoteOption string         `json:"vote,omitempty"`
	Message    string         `json:"message,omitempty"`
	VoteCounts map[string]int `json:"vote_counts,omitempty"`
	SeqNum     int            `json:"seq_num,omitempty"`
}

type Stats struct {
	m sync.Mutex

	sent       int
	confirmed  int
	broadcasts int
	lost       int
	lastSeq    int
}

func (s *Stats) addVote()    { s.m.Lock(); s.sent++; s.m.Unlock() }
func (s *Stats) confirm()    { s.m.Lock(); s.confirmed++; s.m.Unlock() }
func (s *Stats) addBroadcast(){ s.m.Lock(); s.broadcasts++; s.m.Unlock() }

func (s *Stats) seqCheck(n int) {
	s.m.Lock()
	if s.lastSeq > 0 && n > s.lastSeq+1 {
		s.lost += n - s.lastSeq - 1
	}
	s.lastSeq = n
	s.m.Unlock()
}

func (s *Stats) Print() {
	s.m.Lock()
	defer s.m.Unlock()

	fmt.Println("\n===== UDP STATS =====")
	fmt.Println("Votos enviados:", s.sent)
	fmt.Println("Confirmados   :", s.confirmed)
	fmt.Println("Não confirm. :", s.sent-s.confirmed)
	fmt.Println("Broadcasts   :", s.broadcasts)
	fmt.Println("Pacotes perd.:", s.lost)

	total := s.broadcasts + s.lost
	if total > 0 {
		fmt.Printf("Perda estimada: %.2f%%\n", float64(s.lost)/float64(total)*100)
	}
	fmt.Println("=====================\n")
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Uso: go run client.go <nome>")
		return
	}
	name := os.Args[1]
	stats := &Stats{}

	conn, _ := net.Dial("udp", "localhost:9000")
	defer conn.Close()

	go listen(conn, stats)

	send(conn, "REGISTER", name, "")
	fmt.Println("Conectado. Use: VOTE A | STATS | QUIT")

	input := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print(">> ")
		input.Scan()
		cmd := input.Text()

		switch {
		case cmd == "STATS":
			stats.Print()

		case cmd == "QUIT":
			stats.Print()
			return

		case strings.HasPrefix(cmd, "VOTE "):
			option := strings.TrimPrefix(cmd, "VOTE ")
			stats.addVote()
			send(conn, "VOTE", name, option)

		default:
			fmt.Println("Comandos: VOTE <X>, STATS, QUIT")
		}
	}
}

func listen(conn net.Conn, stats *Stats) {
	buf := make([]byte, 4096)
	for {
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		n, err := conn.Read(buf)
		if err != nil { continue }

		var msg Message
		if json.Unmarshal(buf[:n], &msg) != nil { continue }

		switch msg.Type {
		case "ACK":
			stats.confirm()
			fmt.Printf("\n[OK] %s\n>> ", msg.Message)

		case "ERROR":
			fmt.Printf("\n[ERRO] %s\n>> ", msg.Message)

		case "BROADCAST":
			stats.addBroadcast()
			stats.seqCheck(msg.SeqNum)
			fmt.Printf("\n📡 Parcial #%d %v\n>> ", msg.SeqNum, msg.VoteCounts)
		}
	}
}

func send(c net.Conn, t, id, opt string) {
	data,_ := json.Marshal(Message{Type:t, ClientID:id, VoteOption:opt})
	c.Write(data)
}
