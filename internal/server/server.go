package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

// UDPServer gerencia toda a lógica de votação, clientes e comunicação UDP.
type UDPServer struct {
	conn *net.UDPConn // conexão UDP do servidor

	mu sync.Mutex // mutex para evitar race conditions (uso concorrente de maps)

	// Armazena clientes conectados
	// key = ClientID, value = endereço UDP do cliente
	clients map[string]*net.UDPAddr

	// Registro de votos individuais
	// key = ClientID, value = opção votada
	votes map[string]string

	// Contagem total de votos por opção
	// key = opção, value = quantidade de votos
	voteCounts map[string]int

	// Controle do estado da votação
	votingState    VotingState // NotStarted / Active / Ended
	votingDeadline time.Time   // hora em que a votação termina

	// Canal que bufferiza updates para broadcast (evita travar o servidor)
	broadcastChan chan BroadcastUpdate
	broadcastSeq  int // incrementa a cada broadcast para controlar versão
}

///////////////////////////////////////////////////////////////////////////////
// CONSTRUTOR
///////////////////////////////////////////////////////////////////////////////

// NewUDPServer cria uma instância do servidor com as opções disponíveis para votar.
func NewUDPServer(options []string) *UDPServer {
	s := &UDPServer{
		clients:       make(map[string]*net.UDPAddr),
		votes:         make(map[string]string),
		voteCounts:    make(map[string]int),
		votingState:   VotingNotStarted,
		broadcastChan: make(chan BroadcastUpdate, 200), // canal com buffer grande
	}

	// Inicializa contadores das opções
	for _, op := range options {
		s.voteCounts[op] = 0
	}

	// Worker que envia broadcast sempre que houver evento novo
	go s.broadcastWorker()

	return s
}

///////////////////////////////////////////////////////////////////////////////
// INICIAR SERVIDOR
///////////////////////////////////////////////////////////////////////////////

// Start abre o socket UDP e começa a escutar mensagens
func (s *UDPServer) Start(port string) {
	addr, err := net.ResolveUDPAddr("udp", port) // resolve porta
	if err != nil {
		log.Fatal(err)
	}

	s.conn, err = net.ListenUDP("udp", addr) // inicia servidor UDP
	if err != nil {
		log.Fatal(err)
	}
	defer s.conn.Close()

	log.Printf("Servidor UDP ouvindo em %s", port)

	buffer := make([]byte, 4096) // buffer para pacotes recebidos

	// Loop infinito ouvindo clientes
	for {
		n, clientAddr, err := s.conn.ReadFromUDP(buffer)
		if err != nil {
			continue
		}

		// Cria uma cópia do pacote recebido
		data := make([]byte, n)
		copy(data, buffer[:n])
		// Pacote tratado de forma assíncrona (não bloqueia leitura)
		go s.handlePacket(buffer[:n], clientAddr)
	}
}

///////////////////////////////////////////////////////////////////////////////
// ROTEAMENTO DE PACOTES
///////////////////////////////////////////////////////////////////////////////

func (s *UDPServer) handlePacket(data []byte, addr *net.UDPAddr) {
	var msg Message
	if json.Unmarshal(data, &msg) != nil {
		return // ignora pacotes inválidos
	}

	// Roteia pela ação
	switch msg.Type {
	case "REGISTER":
		s.registerClient(msg.ClientID, addr)
	case "VOTE":
		s.processVote(msg.ClientID, msg.VoteOption, addr)
	default:
		log.Println("Mensagem desconhecida:", msg.Type)
	}
}

///////////////////////////////////////////////////////////////////////////////
// REGISTRO DE CLIENTE
///////////////////////////////////////////////////////////////////////////////

func (s *UDPServer) registerClient(id string, addr *net.UDPAddr) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Não permite dois clientes com o mesmo ID
	if _, exists := s.clients[id]; exists {
		s.send(addr, Message{Type: "ERROR", Message: "ID já registrado"})
		return
	}

	// Salva endereço do cliente
	s.clients[id] = addr
	log.Printf("[JOIN] %s (%s)", id, addr)

	// Mensagem padrão
	msg := Message{Type: "ACK", Message: "Aguardando início da votação"}

	// Se já estiver rolando votação, informa tempo restante
	if s.votingState == VotingActive {
		remaining := time.Until(s.votingDeadline).Truncate(time.Second)
		msg.Message = fmt.Sprintf("Votação ativa (%s restantes)", remaining)
	}

	// Se já acabou, manda resultado final
	if s.votingState == VotingEnded {
		msg.Message = fmt.Sprintf("Votação encerrada: %v", s.voteCounts)
	}

	s.send(addr, msg)
}

///////////////////////////////////////////////////////////////////////////////
// PROCESSAMENTO DE VOTO
///////////////////////////////////////////////////////////////////////////////

func (s *UDPServer) processVote(id, option string, addr *net.UDPAddr) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Cliente precisa estar registrado
	if _, ok := s.clients[id]; !ok {
		s.send(addr, Message{Type: "ERROR", Message: "Registre-se primeiro"})
		return
	}

	// Votação precisa estar ativa
	if s.votingState != VotingActive || time.Now().After(s.votingDeadline) {
		s.send(addr, Message{Type: "ERROR", Message: "Votação encerrada"})
		return
	}

	// Não pode votar 2x
	if _, ok := s.votes[id]; ok {
		s.send(addr, Message{Type: "ERROR", Message: "Voto duplicado"})
		return
	}

	// Opção precisa existir
	if _, valid := s.voteCounts[option]; !valid {
		s.send(addr, Message{Type: "ERROR", Message: "Opção inválida"})
		return
	}

	// Registra voto
	s.votes[id] = option
	s.voteCounts[option]++

	// Responde apenas ao votante
	s.send(addr, Message{Type: "ACK", Message: "Voto registrado"})

	// Broadcast para todos verem placar atualizado
	// Agora protegido por mutex
	s.broadcastUpdateLocked()
}

// broadcastUpdateLocked deve ser chamado com o mutex já travado
func (s *UDPServer) broadcastUpdateLocked() {
	s.broadcastSeq++ // incrementa versão do broadcast

	// Cria snapshot seguro dos votos
	snap := make(map[string]int)
	for k, v := range s.voteCounts {
		snap[k] = v
	}

	// Se o canal estiver cheio, descarta (evita travamento)
	select {
	case s.broadcastChan <- BroadcastUpdate{VoteCounts: snap, SeqNum: s.broadcastSeq}:
	default:
		log.Println("[UDP] Broadcast descartado (fila cheia)")
	}
}

// broadcastUpdate faz o lock antes de chamar broadcastUpdateLocked
func (s *UDPServer) broadcastUpdate() {
	s.mu.Lock()
	s.broadcastUpdateLocked()
	s.mu.Unlock()
}

///////////////////////////////////////////////////////////////////////////////
// BROADCAST ASSÍNCRONO
///////////////////////////////////////////////////////////////////////////////

// Worker rodando em goroutine que envia atualizações
func (s *UDPServer) broadcastWorker() {
	for update := range s.broadcastChan {
		s.sendBroadcast(update)
	}
}

// Envia update para todos os clientes
func (s *UDPServer) sendBroadcast(update BroadcastUpdate) {
	data, _ := json.Marshal(Message{
		Type:       "BROADCAST",
		VoteCounts: update.VoteCounts,
		SeqNum:     update.SeqNum,
	})

	s.mu.Lock()
	defer s.mu.Unlock()
	for _, addr := range s.clients {
		// Protege contra escrita em conexão fechada
		if s.conn != nil {
			s.conn.WriteToUDP(data, addr)
		}
	}
}

///////////////////////////////////////////////////////////////////////////////
// FUNÇÃO DE ENVIO INDIVIDUAL
///////////////////////////////////////////////////////////////////////////////

func (s *UDPServer) send(addr *net.UDPAddr, msg Message) {
	data, _ := json.Marshal(msg)
	// Protege contra escrita em conexão fechada
	if s.conn != nil {
		s.conn.WriteToUDP(data, addr)
	}
}

///////////////////////////////////////////////////////////////////////////////
// INICIAR/ENCERRAR VOTAÇÃO
///////////////////////////////////////////////////////////////////////////////

func (s *UDPServer) StartVoting(sec int) {
	s.mu.Lock()

	// Só inicia se ainda não começou
	if s.votingState != VotingNotStarted {
		s.mu.Unlock()
		return
	}

	s.votingState = VotingActive
	s.votingDeadline = time.Now().Add(time.Duration(sec) * time.Second)
	s.mu.Unlock()

	log.Printf("Votação iniciada (%ds)", sec)

	// Anuncia para todos
	s.broadcastUpdate()

	// Agendado encerramento automático
	time.AfterFunc(time.Duration(sec)*time.Second, s.endVoting)
}

func (s *UDPServer) endVoting() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Evita encerrar duas vezes
	if s.votingState != VotingActive {
		return
	}

	s.votingState = VotingEnded
	log.Printf("Votação encerrada: %v", s.voteCounts)

	// Envia resultado final para todos
	s.broadcastUpdateLocked()
}
