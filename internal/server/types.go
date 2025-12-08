package server

// ----------------------------------------------------------
// Estados possíveis da votação
// ----------------------------------------------------------

type VotingState string

const (
	VotingNotStarted VotingState = "NOT_STARTED"
	VotingActive     VotingState = "ACTIVE"
	VotingEnded      VotingState = "ENDED"
)

// ----------------------------------------------------------
// Estrutura do pacote trafegado via UDP
// ----------------------------------------------------------

type Message struct {
	Type       string         `json:"type"`                  // REGISTER | VOTE | BROADCAST | ACK | ERROR
	ClientID   string         `json:"client_id"`             // Identificador único do cliente
	VoteOption string         `json:"vote,omitempty"`        // Enviado em VOTE
	Message    string         `json:"message,omitempty"`     // Respostas do servidor (ACK/ERROR)
	VoteCounts map[string]int `json:"vote_counts,omitempty"` // Usado apenas em BROADCAST
	SeqNum     int            `json:"seq_num,omitempty"`     // Para rastrear perda UDP
}

// ----------------------------------------------------------
// Payload para envio interno ao broadcast worker
// ----------------------------------------------------------

type BroadcastUpdate struct {
	VoteCounts map[string]int // snapshot no momento do voto
	SeqNum     int            // número incremental
}
