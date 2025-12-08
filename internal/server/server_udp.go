package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// UDPServer representa o servidor UDP de votação.
type UDPServer struct {
	// SYSCALL: socket(AF_INET, SOCK_DGRAM, 0) + bind()
	// UDP não usa listen() - socket pronto para recvfrom() imediatamente
	conn *net.UDPConn

	mu         sync.Mutex
	clients    map[string]*net.UDPAddr
	votes      map[string]string
	voteCounts map[string]int

	options           VotingOptions
	useAsyncBroadcast bool
	broadcastChan     chan BroadcastUpdate

	votingState    VotingState
	votingDeadline time.Time

	votesReceived int
	broadcastSeq  int
}

// NewUDPServer cria um novo servidor UDP.
func NewUDPServer(async bool, optionsList []string) *UDPServer {
	s := &UDPServer{
		clients:    make(map[string]*net.UDPAddr),
		votes:      make(map[string]string),
		voteCounts: make(map[string]int),
		options: VotingOptions{
			List:          optionsList,
			DisplayString: fmt.Sprintf("%v", optionsList),
		},
		useAsyncBroadcast: async,
		votingState:       VotingNotStarted,
		broadcastSeq:      0,
		votesReceived:     0,
	}

	for _, op := range optionsList {
		s.voteCounts[op] = 0
	}

	if async {
		s.broadcastChan = make(chan BroadcastUpdate, 1000)
		go s.broadcastWorker()
	}

	return s
}

// Start inicia o servidor UDP.
func (s *UDPServer) Start(port string) {
	// SYSCALL: socket(AF_INET, SOCK_DGRAM, 0)
	// SYSCALL: bind() - associa socket à porta
	addr, err := net.ResolveUDPAddr("udp", port)
	if err != nil {
		log.Fatalf("Erro ao resolver endereço: %v", err)
	}

	s.conn, err = net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("Erro ao iniciar UDP: %v", err)
	}
	defer s.conn.Close()

	log.Printf("Servidor UDP em %s", port)
	log.Printf("Opções: %v", s.options.List)

	buffer := make([]byte, 65536)

	for {
		// SYSCALL: recvfrom() - recebe 1 pacote completo
		n, clientAddr, err := s.conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Erro ao ler: %v", err)
			continue
		}

		go s.handlePacket(buffer[:n], clientAddr)
	}
}

// handlePacket processa um único pacote UDP.
func (s *UDPServer) handlePacket(data []byte, clientAddr *net.UDPAddr) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		log.Printf("Pacote inválido de %s: %v", clientAddr, err)
		return
	}

	switch msg.Type {
	case "REGISTER":
		s.handleRegister(msg.ClientID, clientAddr)

	case "VOTE":
		s.handleVote(msg.ClientID, msg.VoteOption, clientAddr)

	default:
		log.Printf("Tipo de mensagem desconhecido: %s", msg.Type)
	}
}

// handleRegister registra um cliente.
func (s *UDPServer) handleRegister(clientID string, addr *net.UDPAddr) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.clients[clientID]; exists {
		s.sendMessage(addr, Message{
			Type:    "ERROR",
			Message: "ID já em uso",
		})
		return
	}

	s.clients[clientID] = addr
	log.Printf("Cliente registrado: %s (%s)", clientID, addr)

	// Envia status atual
	var statusMsg string
	switch s.votingState {
	case VotingNotStarted:
		statusMsg = "Aguardando início da votação"
	case VotingActive:
		remaining := time.Until(s.votingDeadline).Round(time.Second)
		statusMsg = fmt.Sprintf("Votação ativa! %s restantes. Opções: %v", remaining, s.options.List)
	case VotingEnded:
		statusMsg = fmt.Sprintf("Votação encerrada: %v", s.voteCounts)
	}

	s.sendMessage(addr, Message{
		Type:    "ACK",
		Message: statusMsg,
	})
}

// handleVote processa um voto.
func (s *UDPServer) handleVote(clientID, option string, addr *net.UDPAddr) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.votesReceived++

	if _, exists := s.clients[clientID]; !exists {
		s.sendMessage(addr, Message{Type: "ERROR", Message: "Não registrado"})
		return
	}

	if s.votingState != VotingActive || time.Now().After(s.votingDeadline) {
		s.sendMessage(addr, Message{Type: "ERROR", Message: "Votação inativa"})
		return
	}

	if _, jaVotou := s.votes[clientID]; jaVotou {
		s.sendMessage(addr, Message{Type: "ERROR", Message: "Voto duplicado"})
		return
	}

	valid := false
	for _, opt := range s.options.List {
		if option == opt {
			valid = true
			break
		}
	}
	if !valid {
		s.sendMessage(addr, Message{Type: "ERROR", Message: "Opção inválida"})
		return
	}

	s.votes[clientID] = option
	s.voteCounts[option]++

	log.Printf("Voto %d: %s -> %s", s.votesReceived, clientID, option)

	s.sendMessage(addr, Message{
		Type:    "ACK",
		Message: fmt.Sprintf("Voto: %s", option),
	})

	if s.useAsyncBroadcast {
		s.broadcastAsync()
	} else {
		s.broadcastSync()
	}
}

// broadcastSync envia broadcast de forma bloqueante (demonstra problema de buffer).
func (s *UDPServer) broadcastSync() {
	s.broadcastSeq++

	// CRIA PAYLOAD GRANDE (256KB) para encher buffer UDP rapidamente
	// Isso demonstra: clientes lentos terão buffer cheio e SO descartará pacotes
	padding := make([]byte, 256*1024)
	for i := range padding {
		padding[i] = byte(i % 256)
	}

	msg := Message{
		Type:       "BROADCAST",
		VoteCounts: s.voteCounts,
		SeqNum:     s.broadcastSeq,
		Message:    string(padding), // Payload grande
	}

	data, _ := json.Marshal(msg)

	log.Printf("[SYNC BROADCAST #%d] Enviando %d KB para %d clientes",
		s.broadcastSeq, len(data)/1024, len(s.votes))

	for id := range s.votes {
		if addr, ok := s.clients[id]; ok {
			// SYSCALL: sendto(fd, data, len, flags, &dest_addr, addr_len)
			// UDP não bloqueia por "cliente lento" - mas SO pode descartar
			// se buffer do receptor estiver cheio (SEM NOTIFICAR O SENDER!)
			_, err := s.conn.WriteToUDP(data, addr)
			if err != nil {
				log.Printf("Erro ao enviar para %s: %v", id, err)
			}
		}
	}
}

// broadcastAsync envia broadcast via worker (não bloqueia votações).
func (s *UDPServer) broadcastAsync() {
	s.broadcastSeq++

	snapshot := make(map[string]int)
// broadcastSync envia broadcast bloqueante.
func (s *UDPServer) broadcastSync() {
	s.broadcastSeq++

	// Payload grande (256KB) para demonstrar buffer overflow
	padding := make([]byte, 256*1024)

	msg := Message{
		Type:       "BROADCAST",
		VoteCounts: s.voteCounts,
		SeqNum:     s.broadcastSeq,
// broadcastWorker processa broadcasts assíncronos.
func (s *UDPServer) broadcastWorker() {
	for update := range s.broadcastChan {
		padding := make([]byte, 256*1024)

		msg := Message{
			Type:       "BROADCAST",
			VoteCounts: update.VoteCounts,
			SeqNum:     update.SeqNum,
			Message:    string(padding),
		}

		data, _ := json.Marshal(msg)

		s.mu.Lock()
		targets := make(map[string]*net.UDPAddr)
		for id := range s.votes {
			if addr, ok := s.clients[id]; ok {
				targets[id] = addr
			}
		}
		s.mu.Unlock()

		for _, addr := range targets {
			s.conn.WriteToUDP(data, addr)
		}
	}
// sendMessage envia mensagem JSON.
func (s *UDPServer) sendMessage(addr *net.UDPAddr, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}

	// SYSCALL: sendto() - retorna imediatamente, não espera ACK
	s.conn.WriteToUDP(data, addr)
}
	s.votingState = VotingActive
	s.votingDeadline = time.Now().Add(time.Duration(durationSeconds) * time.Second)

	log.Printf("🗳 VOTAÇÃO INICIADA - Duração: %ds", durationSeconds)

	// Notifica todos os clientes registrados
	announcement := Message{
// StartVoting inicia votação.
func (s *UDPServer) StartVoting(durationSeconds int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.votingState != VotingNotStarted {
		return
	}

	s.votingState = VotingActive
	s.votingDeadline = time.Now().Add(time.Duration(durationSeconds) * time.Second)

	log.Printf("Votação iniciada: %ds", durationSeconds)

	announcement := Message{
		Type:    "BROADCAST",
		Message: fmt.Sprintf("Votação: %d seg. Opções: %v", durationSeconds, s.options.List),
	}

	data, _ := json.Marshal(announcement)

	for _, addr := range s.clients {
		s.conn.WriteToUDP(data, addr)
	}

	time.AfterFunc(time.Duration(durationSeconds)*time.Second, s.endVoting)
}

// endVoting encerra votação.
func (s *UDPServer) endVoting() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.votingState != VotingActive {
		return
	}

	s.votingState = VotingEnded
	log.Printf("Votação encerrada: %v", s.voteCounts)

	result := Message{
		Type:       "BROADCAST",
		Message:    "Encerrado",
		VoteCounts: s.voteCounts,
	}

	data, _ := json.Marshal(result)

	for _, addr := range s.clients {
		s.conn.WriteToUDP(data, addr)
	}
}