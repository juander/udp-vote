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
}

func (s *Stats) AddSent()      { s.m.Lock(); s.sent++; s.m.Unlock() }
func (s *Stats) AddConfirm()   { s.m.Lock(); s.confirmed++; s.m.Unlock() }
func (s *Stats) AddBroadcast() { s.m.Lock(); s.broadcastRecv++; s.m.Unlock() }

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
	fmt.Printf("Pacotes perdidos:    %d\n", s.lost)

	total := s.broadcastRecv + s.lost
	if total > 0 {
		fmt.Printf("Perda estimada:      %.2f%%\n", float64(s.lost)/float64(total)*100)
	}
	fmt.Println("==============================\n")
}

// ========================= Clientes de Teste ==========================

func fastClient(id int, stats *Stats, wg *sync.WaitGroup) {
	defer wg.Done()

	clientID := fmt.Sprintf("FAST_%d", id)
	conn, _ := net.Dial("udp", "localhost:9000")
	if conn == nil { return }
	defer conn.Close()

	send(conn, "REGISTER", clientID, "")
	time.Sleep(80 * time.Millisecond)

	stats.AddSent()
	send(conn, "VOTE", clientID, "A")

	// Lê tudo até expirar — simula cliente ativo
	go readLoop(conn, stats)

	time.Sleep(10 * time.Second)
}

func slowClient(id int, stats *Stats, wg *sync.WaitGroup) {
	defer wg.Done()

	clientID := fmt.Sprintf("SLOW_%d", id)
	conn, _ := net.Dial("udp", "localhost:9000")
	if conn == nil { return }
	defer conn.Close()

	send(conn, "REGISTER", clientID, "")
	time.Sleep(120 * time.Millisecond)

	stats.AddSent()
	send(conn, "VOTE", clientID, "B")

	// Lê apenas uma resposta e "morre" => gera perda
	buffer := make([]byte, 2048)
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	n, _ := conn.Read(buffer)

	var msg Message
	if json.Unmarshal(buffer[:n], &msg) == nil && msg.Type == "ACK" {
		stats.AddConfirm()
	}

	fmt.Printf("[SLOW_%d] desconectou (gerando perda)\n", id)
	time.Sleep(10 * time.Second)
}

// recebe pacotes e atualiza estatísticas
func readLoop(conn net.Conn, stats *Stats) {
	buf := make([]byte, 4096)
	for {
		conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		n, err := conn.Read(buf)
		if err != nil { return }

		var msg Message
		if json.Unmarshal(buf[:n], &msg) != nil { continue }

		switch msg.Type {
		case "ACK":
			stats.AddConfirm()

		case "BROADCAST":
			stats.AddBroadcast()
			stats.SeqCheck(msg.SeqNum)
		}
	}
}

// envia pacote UDP
func send(conn net.Conn, t, id, vote string) {
	data,_ := json.Marshal(Message{Type:t, ClientID:id, VoteOption:vote})
	conn.Write(data)
}

// =========================== MAIN TEST ================================
func main() {
	stats := &Stats{}
	var wg sync.WaitGroup

	fmt.Println("==== TESTE UDP ====")
	fmt.Println("FAST → recebe broadcast")
	fmt.Println("SLOW → desconecta e causa perda")
	time.Sleep(1 * time.Second)

	// clientes lentos causam perda
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go slowClient(i, stats, &wg)
	}

	time.Sleep(500 * time.Millisecond)

	// carga real
	for i := 0; i < 25; i++ {
		wg.Add(1)
		go fastClient(i, stats, &wg)
		time.Sleep(20 * time.Millisecond)
	}

	wg.Wait()
	stats.Print()
}
