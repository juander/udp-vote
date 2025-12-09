
# UDP Voting

## Overview
The **UDP Voting** project demonstrates a voting system using the **User Datagram Protocol (UDP)**. Unlike **TCP**, UDP is connectionless and does not guarantee message delivery or the order of delivery, which makes it an interesting choice for this voting system. This project highlights the **Ghost Vote** phenomenon, where votes can be counted more than once due to issues like packet loss and retransmission.

### Key Concepts
- **UDP vs TCP**: Unlike TCP, which is reliable and guarantees delivery of packets in order, UDP does not. In UDP, packets can be lost, duplicated, or arrive out of order, which may cause votes to be counted multiple times (a "Ghost Vote").
- **ACK Usage**: The typical **ACK** (Acknowledgement) message, which is generally used to confirm message receipt in protocols like TCP, is implemented here in a non-traditional manner. Instead of a simple ACK, the server sends specific acknowledgment messages (`ACK`, `ERROR`, etc.) based on the current state of the system (e.g., "Vote Registered", "Duplicate Vote", "Vote Closed").
- **Ghost Vote**: A phenomenon observed in UDP-based systems, where repeated or missed packets can cause the same vote to be counted more than once. This project demonstrates how packet loss can lead to erroneous voting results.

## Features
- **UDP-based Voting**: Clients send votes to the server using the unreliable UDP protocol.
- **Ghost Vote Simulation**: Demonstrates how packet loss and retransmission issues can cause votes to be counted more than once.
- **Broadcasting Results**: The server broadcasts the current vote counts to all connected clients, ensuring everyone is updated.
- **Packet Loss Testing**: Includes tests to simulate UDP packet loss and verify system behavior under those conditions.
- **Concurrency Management**: Uses `sync.Mutex` to prevent race conditions when accessing shared data structures (like votes and clients).
- **Voting States**: The voting process has different states, including `NOT_STARTED`, `ACTIVE`, and `ENDED`. The system transitions between these states, with different behaviors depending on the current state (e.g., no voting allowed when the vote is closed).

## Project Structure
```
udp-vote
├── cmd
│   ├── client
│   │   └── main.go          # Entry point for the UDP voting client
│   └── server
│       └── main.go          # Entry point for the UDP voting server
├── docs
│   ├── architecture.md      # Architecture overview of the UDP voting system
│   └── udp-internals.md     # Explanation of UDP internals and implications for voting
├── internal
│   └── server
│       └── server.go        # Implementation of the UDP voting server logic
├── logs
│   └── .gitkeep             # Keeps the logs directory tracked in version control
├── test
│   └── packet_loss_test.go  # Tests for simulating packet loss scenarios
├── go.mod                   # Module definition and dependencies
├── .gitignore               # Files and directories to ignore in version control
└── README.md                # Project documentation
```

## Detailed Description

### **Server (UDPServer)**

The server manages the voting process, the registration of clients, vote counting, and broadcasting results. It listens for incoming UDP packets, processes the messages, and updates the vote counts. It handles different states of the voting process, including the start, active voting, and end of the voting.

#### **Server Features**
1. **Registration of Clients**: Clients register with the server using a unique `ClientID`. Once registered, they can vote. If a client tries to register with an already existing ID, the server returns an error.
2. **Vote Processing**: The server processes votes by ensuring that:
   - The client is registered.
   - The vote is placed during an active voting session.
   - The client has not already voted.
   - The chosen voting option is valid.
3. **Broadcasting Results**: After every vote, the server broadcasts the updated vote counts to all connected clients. This broadcast happens asynchronously to avoid blocking the main process.
4. **Concurrency and Mutex**: To prevent race conditions when accessing the shared data structures (`clients`, `votes`, `voteCounts`), the server uses a `sync.Mutex`.
5. **Non-traditional ACK**: The server sends specific acknowledgment messages:
   - **ACK**: Confirms successful registration or voting.
   - **ERROR**: Informs the client of an error (e.g., invalid vote, duplicate vote).
6. **State Transitions**: The server handles three main voting states:
   - **NOT_STARTED**: Voting has not yet begun.
   - **ACTIVE**: Voting is ongoing.
   - **ENDED**: Voting has concluded, and no further votes are accepted.

### **Client (UDP Client)**

The client connects to the server, registers, and submits votes. The client also receives acknowledgments and updates from the server.

#### **Client Features**
1. **Registration**: The client first registers with the server using a unique `ClientID`. Upon successful registration, the client is informed about the current voting status (whether voting is open or closed).
2. **Vote Submission**: Once registered, the client can vote by sending a vote option to the server. The client can only vote once, and the server ensures that duplicate votes are not accepted.
3. **ACK and Error Handling**: The client listens for ACKs and error messages from the server. It will display the relevant messages, such as "Vote Registered" or "Vote Closed".
4. **Statistics**: The client tracks UDP statistics such as:
   - Votes sent
   - Votes confirmed by the server
   - Broadcasts received
   - Lost packets (due to UDP’s unreliable nature)
   The client can print these statistics to the console for debugging and testing purposes.

### **UDP Communication**
The server and client communicate using UDP packets. These packets are serialized in **JSON** format, which allows for flexible and easy-to-parse messages.

- **REGISTER**: Sent by the client to register with the server.
- **VOTE**: Sent by the client to cast a vote.
- **BROADCAST**: Sent by the server to broadcast the current vote counts to all clients.
- **ACK**: Sent by the server to acknowledge a successful registration or vote.
- **ERROR**: Sent by the server to indicate an error (e.g., duplicate vote, invalid vote).

### **Ghost Vote Simulation**
Due to the nature of UDP, packets can be lost, duplicated, or delayed. The project includes a **Ghost Vote Simulation** feature to show how these issues might lead to the same vote being counted multiple times. When a packet is lost, it is possible for a vote to be resubmitted or counted twice if the client retransmits the packet or if the server misinterprets a retransmitted packet as a new vote.

### **Packet Loss Testing**
The project includes tests to simulate UDP packet loss. These tests allow developers to simulate real-world network conditions and verify how the system behaves when packets are lost or duplicated.

- **Simulating Packet Loss**: The test suite includes a test that intentionally drops packets, mimicking real-world network conditions where some UDP packets do not reach their destination.
- **Behavior Verification**: The tests verify how the server handles lost or duplicate packets and how this affects the vote counts.

## Setup Instructions
To set up and run the project, follow these steps:

1. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd udp-vote
   ```

2. **Install dependencies**:
   ```bash
   go mod tidy
   ```

3. **Run the server**:
   ```bash
   go run cmd/server/main.go
   ```

4. **Run the client**:
   ```bash
   go run cmd/client/main.go
   ```

## Usage
1. **Start the server**: Run the server first to listen for incoming votes.
2. **Start one or more clients**: Run the client to send votes to the server.
3. **Observe Results**: Watch for **Ghost Votes** and other behaviors induced by packet loss.

### Example:
1. Start the server:
   ```bash
   go run cmd/server/main.go
   ```
2. Start a client and register:
   ```bash
   go run cmd/client/main.go
   ```
3. The client will be registered and can vote with the following command:
   ```bash
   VOTE A
   ```

## Conclusion
This project serves as an **educational tool** to understand the challenges and implications of using **UDP** for a **voting system**. It highlights:
- The potential for **erroneous vote counting** due to UDP's unreliability.
- The implementation of a non-traditional **ACK** mechanism for handling registrations and votes.
- The simulation and testing of **Ghost Votes** and **packet loss** in a real-world UDP-based system.

By experimenting with this system, users can gain valuable insights into the behavior of UDP in distributed systems, especially in critical applications like voting, where data integrity is paramount.
