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

type ClientStats struct {
	mu                sync.Mutex
	votesSent         int
	votesConfirmed    int
	broadcastsReceived int
	lastSeqNum        int
	packetsLost       int
}

func (s *ClientStats) PrintStats() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	fmt.Println("\n--- Estatísticas UDP ---")
	fmt.Printf("Votos enviados:     %d\n", s.votesSent)
	fmt.Printf("Votos confirmados:  %d\n", s.votesConfirmed)
	
	if s.votesSent > s.votesConfirmed {
		fmt.Printf("Votos fantasma:     %d\n", s.votesSent-s.votesConfirmed)
	}
	
	fmt.Printf("Broadcasts:         %d\n", s.broadcastsReceived)
	fmt.Printf("Packets perdidos:   %d\n", s.packetsLost)
	
	if s.packetsLost > 0 {
		loss := float64(s.packetsLost) / float64(s.broadcastsReceived+s.packetsLost) * 100
		fmt.Printf("Taxa de perda:      %.1f%%\n", loss)
	}
	fmt.Println()
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Uso: ./client <nome>")
		return
	}

	clientID := os.Args[1]
	stats := &ClientStats{}

	// SYSCALL: socket(AF_INET, SOCK_DGRAM, 0)
	conn, err := net.Dial("udp", "localhost:9000")
	if err != nil {
		fmt.Println("Erro:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Conectado ao servidor UDP")
	go receiveMessages(conn, stats)

	sendMessage(conn, Message{Type: "REGISTER", ClientID: clientID})

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print(">> ")
		if !scanner.Scan() {
			break
		}

		text := strings.TrimSpace(scanner.Text())

		if text == "QUIT" {
			stats.PrintStats()
			break
		}

		if text == "STATS" {
			stats.PrintStats()
			continue
		}

		if strings.HasPrefix(text, "VOTE ") {
			option := strings.TrimPrefix(text, "VOTE ")
			
			stats.mu.Lock()
			stats.votesSent++
			stats.mu.Unlock()
			
			sendMessage(conn, Message{
				Type:       "VOTE",
				ClientID:   clientID,
				VoteOption: option,
			})
			continue
		}
	}
}

func receiveMessages(conn net.Conn, stats *ClientStats) {
	buffer := make([]byte, 65536)

	for {
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		
		n, err := conn.Read(buffer)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			return
		}

		var msg Message
		if json.Unmarshal(buffer[:n], &msg) != nil {
			continue
		}

		stats.mu.Lock()
		switch msg.Type {
		case "ACK":
			if strings.Contains(msg.Message, "Voto") {
				stats.votesConfirmed++
			}
			stats.mu.Unlock()
			fmt.Printf("\n[ACK] %s\n>> ", msg.Message)

		case "ERROR":
			stats.mu.Unlock()
			fmt.Printf("\n[ERRO] %s\n>> ", msg.Message)

		case "BROADCAST":
			stats.broadcastsReceived++
			
			if stats.lastSeqNum > 0 && msg.SeqNum > stats.lastSeqNum+1 {
				gap := msg.SeqNum - stats.lastSeqNum - 1
				stats.packetsLost += gap
				stats.mu.Unlock()
				fmt.Printf("\n[GAP] %d packets perdidos (seq %d -> %d)\n>> ",
					gap, stats.lastSeqNum, msg.SeqNum)
			} else {
				stats.mu.Unlock()
			}
			
			stats.mu.Lock()
			stats.lastSeqNum = msg.SeqNum
			stats.mu.Unlock()
			
			if msg.VoteCounts != nil {
				fmt.Printf("\n[#%d] %v\n>> ", msg.SeqNum, msg.VoteCounts)
			}

		default:
			stats.mu.Unlock()
		}
	}
}

func sendMessage(conn net.Conn, msg Message) {
	data, _ := json.Marshal(msg)
	// SYSCALL: sendto() - retorna imediatamente
	conn.Write(data)
}
