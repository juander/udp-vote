package main

import (
	"encoding/json"
	"fmt"
	"net"
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
	mu              sync.Mutex
	votesSent       int
	votesConfirmed  int
	broadcastsRecv  int
	packetsLost     int
}

func (s *Stats) Print() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	fmt.Println("\n=== RESULTADOS ===")
	fmt.Printf("Votos enviados:     %d\n", s.votesSent)
	fmt.Printf("Votos confirmados:  %d\n", s.votesConfirmed)
	fmt.Printf("Votos fantasma:     %d\n", s.votesSent-s.votesConfirmed)
	fmt.Printf("Broadcasts receb:   %d\n", s.broadcastsRecv)
	fmt.Printf("Packets perdidos:   %d\n", s.packetsLost)
	
	if total := s.broadcastsRecv + s.packetsLost; total > 0 {
		loss := float64(s.packetsLost) / float64(total) * 100
		fmt.Printf("Taxa de perda:      %.1f%%\n", loss)
	}
}

func fastClient(id int, stats *Stats, wg *sync.WaitGroup) {
	defer wg.Done()

	conn, _ := net.Dial("udp", "localhost:9000")
	if conn == nil {
		return
	}
	defer conn.Close()

	clientID := fmt.Sprintf("FAST_%d", id)
	
	send(conn, Message{Type: "REGISTER", ClientID: clientID})
	time.Sleep(100 * time.Millisecond)

	stats.mu.Lock()
	stats.votesSent++
	stats.mu.Unlock()

	send(conn, Message{Type: "VOTE", ClientID: clientID, VoteOption: "A"})

	go func() {
		buffer := make([]byte, 65536)
		for {
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
			n, err := conn.Read(buffer)
			if err != nil {
				return
			}

			var msg Message
			if json.Unmarshal(buffer[:n], &msg) == nil {
				if msg.Type == "ACK" && msg.Message == "Voto: A" {
					stats.mu.Lock()
					stats.votesConfirmed++
					stats.mu.Unlock()
				}
				if msg.Type == "BROADCAST" && msg.SeqNum > 0 {
					stats.mu.Lock()
					stats.broadcastsRecv++
					stats.mu.Unlock()
				}
			}
		}
	}()

	time.Sleep(30 * time.Second)
}

func slowClient(id int, stats *Stats, wg *sync.WaitGroup) {
	defer wg.Done()

	conn, _ := net.Dial("udp", "localhost:9000")
	if conn == nil {
		return
	}
	defer conn.Close()

	clientID := fmt.Sprintf("SLOW_%d", id)
	
	send(conn, Message{Type: "REGISTER", ClientID: clientID})
	time.Sleep(100 * time.Millisecond)

	stats.mu.Lock()
	stats.votesSent++
	stats.mu.Unlock()

	send(conn, Message{Type: "VOTE", ClientID: clientID, VoteOption: "B"})

	buffer := make([]byte, 65536)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, _ := conn.Read(buffer)
	
	var msg Message
	if json.Unmarshal(buffer[:n], &msg) == nil && msg.Type == "ACK" {
		stats.mu.Lock()
		stats.votesConfirmed++
		stats.mu.Unlock()
	}

	fmt.Printf("[SLOW_%d] Parou de ler\n", id)
	time.Sleep(30 * time.Second)
}

func send(conn net.Conn, msg Message) {
	data, _ := json.Marshal(msg)
	conn.Write(data)
}

func main() {
	var wg sync.WaitGroup
	stats := &Stats{}

	fmt.Println("=== TESTE UDP ===")
	fmt.Println("Demonstra: votos fantasma + buffer overflow")
	time.Sleep(2 * time.Second)

	// 5 clientes lentos (buffer overflow)
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go slowClient(i, stats, &wg)
		time.Sleep(50 * time.Millisecond)
	}

	time.Sleep(1 * time.Second)

	// 30 clientes rápidos
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go fastClient(i, stats, &wg)
		time.Sleep(20 * time.Millisecond)
	}

	fmt.Println("\nTeste rodando (30s)...")
	wg.Wait()

	stats.Print()
}
