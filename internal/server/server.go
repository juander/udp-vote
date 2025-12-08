package server

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type VotingState string

const (
	VotingNotStarted VotingState = "NOT_STARTED"
	VotingActive     VotingState = "ACTIVE"
	VotingEnded      VotingState = "ENDED"
)

// VotingOptions encapsulates the available voting options.
type VotingOptions struct {
	List          []string
	DisplayString string
}

// Server represents the UDP voting server.
type Server struct {
	// Mutex protects concurrent access to shared maps
	mu         sync.Mutex
	clients    map[string]net.UDPAddr // Map of active client addresses
	votes      map[string]string       // Vote history
	voteCounts map[string]int          // Aggregate score

	options           VotingOptions
	votingState       VotingState
	votingDeadline    time.Time
	broadcastInterval time.Duration
}

// NewServer initializes the server with voting options.
func NewServer(optionsList []string, broadcastInterval time.Duration) *Server {
	s := &Server{
		clients:    make(map[string]net.UDPAddr),
		votes:      make(map[string]string),
		voteCounts: make(map[string]int),
		options: VotingOptions{
			List:          optionsList,
			DisplayString: strings.Join(optionsList, ", "),
		},
		votingState:       VotingNotStarted,
		broadcastInterval: broadcastInterval,
	}

	// Initialize counters for all options
	for _, op := range optionsList {
		s.voteCounts[op] = 0
	}

	return s
}

// Start begins listening for incoming votes on the specified port.
func (s *Server) Start(port string) {
	addr, err := net.ResolveUDPAddr("udp", port)
	if err != nil {
		log.Fatalf("Failed to resolve address: %v", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on UDP: %v", err)
	}
	defer conn.Close()

	log.Printf("UDP Voting Server listening on %s", port)

	go s.broadcastUpdates(conn)

	buffer := make([]byte, 1024)
	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Error reading from UDP: %v", err)
			continue
		}

		go s.handleVote(conn, clientAddr, buffer[:n])
	}
}

// handleVote processes incoming votes from clients.
func (s *Server) handleVote(conn *net.UDPConn, clientAddr *net.UDPAddr, vote []byte) {
	id := strings.TrimSpace(string(vote))
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.votingState == VotingNotStarted {
		return
	}

	// Simulate Ghost Vote phenomenon
	if _, exists := s.votes[id]; !exists {
		s.votes[id] = string(vote)
		s.voteCounts[string(vote)]++
		log.Printf("Vote accepted from %s: %s", clientAddr.String(), string(vote))
	} else {
		log.Printf("Ghost Vote detected from %s: %s", clientAddr.String(), string(vote))
	}

	s.clients[id] = *clientAddr
}

// broadcastUpdates periodically sends the current voting status to all clients.
func (s *Server) broadcastUpdates(conn *net.UDPConn) {
	ticker := time.NewTicker(s.broadcastInterval)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		if s.votingState == VotingActive {
			message := fmt.Sprintf("Current Votes: %v\n", s.voteCounts)
			for _, addr := range s.clients {
				_, err := conn.WriteToUDP([]byte(message), &addr)
				if err != nil {
					log.Printf("Error broadcasting to %s: %v", addr.String(), err)
				}
			}
		}
		s.mu.Unlock()
	}
}

// StartVoting initiates the voting process.
func (s *Server) StartVoting(durationSeconds int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.votingState != VotingNotStarted {
		return
	}

	s.votingState = VotingActive
	s.votingDeadline = time.Now().Add(time.Duration(durationSeconds) * time.Second)

	log.Printf("Voting started. Deadline: %s", s.votingDeadline.Format("15:04:05"))

	// Timer to automatically end voting
	time.AfterFunc(time.Duration(durationSeconds)*time.Second, func() {
		s.endVoting()
	})
}

// endVoting ends the voting and sends final results.
func (s *Server) endVoting() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.votingState != VotingActive {
		return
	}

	s.votingState = VotingEnded
	log.Println("Voting ended")

	// Final results
	result := fmt.Sprintf("Voting ended. Final results: %v\n", s.voteCounts)
	for _, addr := range s.clients {
		// Send final results to all clients
		// Note: Broadcasting final results can be added here if needed
	}
}