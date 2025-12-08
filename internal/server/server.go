package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

type UDPServer struct {
	conn *net.UDPConn

	mu         sync.Mutex
	clients    map[string]*net.UDPAddr // ID → endereço
	votes      map[string]string       // ID → voto
	voteCounts map[string]int          // opção → total

	votingState    VotingState
	votingDeadline time.Time

	broadcastChan chan BroadcastUpdate
	broadcastSeq  int
}

// ============================================================================
// INIT
// ============================================================================

func NewUDPServer(options []string) *UDPServer {
	s := &UDPServer{
		clients:       make(map[string]*net.UDPAddr),
		votes:         make(map[string]string),
		voteCounts:    make(map[string]int),
		votingState:   VotingNotStarted,
		broadcastChan: make(chan BroadcastUpdate, 200),
	}

	for _, op := range options {
		s.voteCounts[op] = 0
	}

	// sempre async agora
	go s.broadcastWorker()

	return s
}

// ============================================================================
// START
// ============================================================================

func (s *UDPServer) Start(port string) {
	addr, err := net.ResolveUDPAddr("udp", port)
	if err != nil { log.Fatal(err) }

	s.conn, err = net.ListenUDP("udp", addr)
	if err != nil { log.Fatal(err) }
	defer s.conn.Close()

	log.Printf("Servidor UDP ouvindo em %s", port)

	buffer := make([]byte, 4096)

	for {
		n, clientAddr, err := s.conn.ReadFromUDP(buffer)
		if err != nil { continue }
		go s.handlePacket(buffer[:n], clientAddr)
	}
}

// ============================================================================
// HANDLE PACKETS
// ============================================================================

func (s *UDPServer) handlePacket(data []byte, addr *net.UDPAddr) {
	var msg Message
	if json.Unmarshal(data, &msg) != nil {
		return
	}

	switch msg.Type {
	case "REGISTER": s.registerClient(msg.ClientID, addr)
	case "VOTE":     s.processVote(msg.ClientID, msg.VoteOption, addr)
	default:
		log.Println("Mensagem desconhecida:", msg.Type)
	}
}

// ============================================================================
// REGISTER
// ============================================================================

func (s *UDPServer) registerClient(id string, addr *net.UDPAddr) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.clients[id]; exists {
		s.send(addr, Message{Type:"ERROR", Message:"ID já registrado"})
		return
	}

	s.clients[id] = addr
	log.Printf("[JOIN] %s (%s)", id, addr)

	msg := Message{Type:"ACK", Message:"Aguardando início da votação"}

	if s.votingState == VotingActive {
		remaining := time.Until(s.votingDeadline).Truncate(time.Second)
		msg.Message = fmt.Sprintf("Votação ativa (%s restantes)", remaining)
	}
	if s.votingState == VotingEnded {
		msg.Message = fmt.Sprintf("Votação encerrada: %v", s.voteCounts)
	}

	s.send(addr, msg)
}

// ============================================================================
// VOTE
// ============================================================================

func (s *UDPServer) processVote(id, option string, addr *net.UDPAddr) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clients[id]; !ok {
		s.send(addr, Message{Type:"ERROR", Message:"Registre-se primeiro"})
		return
	}

	if s.votingState != VotingActive || time.Now().After(s.votingDeadline) {
		s.send(addr, Message{Type:"ERROR", Message:"Votação encerrada"})
		return
	}

	if _, ok := s.votes[id]; ok {
		s.send(addr, Message{Type:"ERROR", Message:"Voto duplicado"})
		return
	}

	if _, valid := s.voteCounts[option]; !valid {
		s.send(addr, Message{Type:"ERROR", Message:"Opção inválida"})
		return
	}

	s.votes[id] = option
	s.voteCounts[option]++

	s.send(addr, Message{Type:"ACK", Message:"Voto registrado"})
	s.broadcastUpdate()
}

// ============================================================================
// BROADCAST (assíncrono sempre)
// ============================================================================

func (s *UDPServer) broadcastUpdate() {
	s.broadcastSeq++

	snap := make(map[string]int)
	for k,v := range s.voteCounts { snap[k] = v }

	select {
	case s.broadcastChan <- BroadcastUpdate{VoteCounts:snap, SeqNum:s.broadcastSeq}:
	default:
		log.Println("[UDP] Broadcast descartado (fila cheia)")
	}
}

func (s *UDPServer) broadcastWorker() {
	for update := range s.broadcastChan {
		s.sendBroadcast(update)
	}
}

func (s *UDPServer) sendBroadcast(update BroadcastUpdate) {
	data,_ := json.Marshal(Message{
		Type:"BROADCAST",
		VoteCounts:update.VoteCounts,
		SeqNum:update.SeqNum,
	})

	s.mu.Lock()
	for _, addr := range s.clients {
		s.conn.WriteToUDP(data, addr)
	}
	s.mu.Unlock()
}

// ============================================================================
// SEND
// ============================================================================

func (s *UDPServer) send(addr *net.UDPAddr, msg Message) {
	data,_ := json.Marshal(msg)
	s.conn.WriteToUDP(data, addr)
}

// ============================================================================
// VOTING CONTROL
// ============================================================================

func (s *UDPServer) StartVoting(sec int) {
	s.mu.Lock()
	if s.votingState != VotingNotStarted {
		s.mu.Unlock()
		return
	}

	s.votingState = VotingActive
	s.votingDeadline = time.Now().Add(time.Duration(sec)*time.Second)
	s.mu.Unlock()

	log.Printf("Votação iniciada (%ds)", sec)
	s.broadcastUpdate()

	time.AfterFunc(time.Duration(sec)*time.Second, s.endVoting)
}

func (s *UDPServer) endVoting() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.votingState != VotingActive { return }

	s.votingState = VotingEnded
	log.Printf("Votação encerrada: %v", s.voteCounts)
	s.broadcastUpdate()
}
