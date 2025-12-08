package test

import (
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"
)

const (
	serverAddr = "localhost:9000"
	numClients = 10
	numVotes   = 100
	packetLossThreshold = 0.1 // 10% packet loss
)

// simulatePacketLoss simulates packet loss by randomly dropping packets.
func simulatePacketLoss(vote string) bool {
	return rand.Float64() > packetLossThreshold
}

// TestPacketLoss simulates packet loss scenarios in the UDP voting system.
func TestPacketLoss(t *testing.T) {
	// Start the UDP server
	go startUDPServer()

	time.Sleep(1 * time.Second) // Give the server time to start

	var wg sync.WaitGroup
	votesReceived := make(map[string]int)

	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go func(clientID string) {
			defer wg.Done()
			conn, err := net.Dial("udp", serverAddr)
			if err != nil {
				t.Errorf("Error connecting to server: %v", err)
				return
			}
			defer conn.Close()

			for j := 0; j < numVotes; j++ {
				vote := fmt.Sprintf("VOTE %s from client %s", "A", clientID)
				if !simulatePacketLoss(vote) {
					_, err := conn.Write([]byte(vote))
					if err != nil {
						t.Errorf("Error sending vote: %v", err)
					}
				}
			}
		}(fmt.Sprintf("Client%d", i))
	}

	wg.Wait()

	// Check the results
	time.Sleep(2 * time.Second) // Wait for server to process votes

	// Here you would typically check the state of the server to see how many votes were counted
	// This is a placeholder for the actual verification logic
	t.Log("Packet loss simulation completed.")
}

// startUDPServer starts a simple UDP server for testing.
func startUDPServer() {
	addr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		fmt.Println("Error resolving address:", err)
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("Error starting UDP server:", err)
		return
	}
	defer conn.Close()

	buffer := make([]byte, 1024)

	for {
		n, clientAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error reading from UDP:", err)
			continue
		}

		vote := string(buffer[:n])
		fmt.Printf("Received vote: %s from %s\n", vote, clientAddr)

		// Here you would process the vote and broadcast the results to clients
	}
}