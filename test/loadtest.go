package main

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"
)

// =============== Estruturas comuns ao cliente/servidor ================

type Message struct {
	Type       string         `json:"type"`
	ClientID   string         `json:"client_id"`
	VoteOption string         `json:"vote,omitempty"`
	Message    string         `json:"message,omitempty"`
	VoteCounts map[string]int `json:"vote_counts,omitempty"`
	SeqNum     int            `json:"seq_num,omitempty"`
}

// ============================ Estatísticas ============================

type Stats struct {
	m             sync.Mutex
	sent          int
	confirmed     int
	broadcastRecv int
	lost          int
	lastSeq       int
	broadcastFail int
}

func (s *Stats) AddSent()          { s.m.Lock(); s.sent++; s.m.Unlock() }
func (s *Stats) AddConfirm()       { s.m.Lock(); s.confirmed++; s.m.Unlock() }
func (s *Stats) AddBroadcast()     { s.m.Lock(); s.broadcastRecv++; s.m.Unlock() }
func (s *Stats) AddBroadcastFail() { s.m.Lock(); s.broadcastFail++; s.m.Unlock() }

func (s *Stats) SeqCheck(n int) {
	s.m.Lock()
	if s.lastSeq > 0 && n > s.lastSeq+1 {
		s.lost += (n - s.lastSeq - 1)
	}
	s.lastSeq = n
	s.m.Unlock()
}

func (s *Stats) Print() {
	s.m.Lock()
	defer s.m.Unlock()

	fmt.Println("\n====== RESULTADOS UDP ======")
	fmt.Printf("Votos enviados:      %d\n", s.sent)
	fmt.Printf("Confirmados (ACK):   %d\n", s.confirmed)
	fmt.Printf("Votos fantasma:      %d\n", s.sent-s.confirmed)
	fmt.Printf("Broadcasts recebidos:%d\n", s.broadcastRecv)
	fmt.Printf("Broadcasts perdidos: %d\n", s.broadcastFail)
	fmt.Printf("Pacotes perdidos:    %d\n", s.lost)

	total := s.broadcastRecv + s.lost
	if total > 0 {
		fmt.Printf("Perda estimada:      %.2f%%\n", float64(s.lost)/float64(total)*100)
	}

	fmt.Println("==============================\n")
}

// ========================= Clientes de Teste ==========================

func activeClient(id int, stats *Stats, wg *sync.WaitGroup) {
	defer wg.Done()
	clientID := fmt.Sprintf("ACTIVE_%d", id)

	conn, err := net.Dial("udp", "localhost:9000")
	if err != nil {
		fmt.Printf("[ACTIVE_%d] erro ao conectar\n", id)
		return
	}
	defer conn.Close()

	ackCh := make(chan struct{})
	go func() {
		buf := make([]byte, 2048)
		for {
			conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			n, err := conn.Read(buf)
			if err != nil {
				return
			}
			var msg Message
			if json.Unmarshal(buf[:n], &msg) != nil {
				continue
			}
			if msg.Type == "ACK" {
				ackCh <- struct{}{}
				return
			}
		}
	}()

	send(conn, "REGISTER", clientID, "")
	select {
	case <-ackCh:
	case <-time.After(2 * time.Second):
		fmt.Printf("[ACTIVE_%d] não recebeu ACK de registro\n", id)
		return
	}

	time.Sleep(80 * time.Millisecond)
	stats.AddSent()
	send(conn, "VOTE", clientID, "A")

	broadcasts := make(map[int]bool)
	end := time.Now().Add(10 * time.Second)
	buf := make([]byte, 4096)

	for time.Now().Before(end) {
		conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, err := conn.Read(buf)
		if err != nil {
			continue
		}
		var msg Message
		if json.Unmarshal(buf[:n], &msg) != nil {
			continue
		}
		switch msg.Type {
		case "ACK":
			if msg.Message == "Voto registrado" {
				stats.AddConfirm()
			}
		case "BROADCAST":
			stats.AddBroadcast()
			stats.SeqCheck(msg.SeqNum)
			broadcasts[msg.SeqNum] = true
		}
	}

	if len(broadcasts) == 0 {
		stats.AddBroadcastFail()
		fmt.Printf("[ACTIVE_%d] não recebeu nenhum broadcast!\n", id)
	}
}

func inactiveClient(id int, stats *Stats, wg *sync.WaitGroup) {
	defer wg.Done()
	clientID := fmt.Sprintf("INACTIVE_%d", id)

	conn, err := net.Dial("udp", "localhost:9000")
	if err != nil {
		fmt.Printf("[INACTIVE_%d] erro ao conectar\n", id)
		return
	}
	defer conn.Close()

	ackCh := make(chan struct{})
	go func() {
		buf := make([]byte, 2048)
		for {
			conn.SetReadDeadline(time.Now().Add(2 * time.Second))
			n, err := conn.Read(buf)
			if err != nil {
				return
			}
			var msg Message
			if json.Unmarshal(buf[:n], &msg) != nil {
				continue
			}
			if msg.Type == "ACK" {
				ackCh <- struct{}{}
				return
			}
		}
	}()

	send(conn, "REGISTER", clientID, "")
	select {
	case <-ackCh:
	case <-time.After(2 * time.Second):
		fmt.Printf("[INACTIVE_%d] não recebeu ACK de registro\n", id)
		return
	}

	time.Sleep(120 * time.Millisecond)
	stats.AddSent()
	send(conn, "VOTE", clientID, "B")

	buffer := make([]byte, 2048)
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	n, _ := conn.Read(buffer)

	var msg Message
	if json.Unmarshal(buffer[:n], &msg) == nil && msg.Type == "ACK" && msg.Message == "Voto registrado" {
		stats.AddConfirm()
	}

	fmt.Printf("[INACTIVE_%d] cliente ficou inativo (não recebe mais mensagens)\n", id)
	time.Sleep(10 * time.Second)
}

func send(conn net.Conn, t, id, vote string) {
	data, _ := json.Marshal(Message{Type: t, ClientID: id, VoteOption: vote})
	conn.Write(data)
}

// =========================== MAIN TEST ================================

func main() {
	stats := &Stats{}
	var wg sync.WaitGroup

	fmt.Println("==== TESTE UDP ====")
	fmt.Println("ACTIVE → recebe broadcast")
	fmt.Println("INACTIVE → desconecta e não recebe broadcast")

	time.Sleep(1 * time.Second)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go inactiveClient(i, stats, &wg)
	}

	time.Sleep(500 * time.Millisecond)

	for i := 0; i < 25; i++ {
		wg.Add(1)
		go activeClient(i, stats, &wg)
		time.Sleep(20 * time.Millisecond)
	}

	wg.Wait()
	stats.Print()
}
