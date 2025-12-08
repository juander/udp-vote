package server

// VotingState representa o estado da votação.
type VotingState string

const (
	VotingNotStarted VotingState = "NOT_STARTED"
	VotingActive     VotingState = "ACTIVE"
	VotingEnded      VotingState = "ENDED"
)

// VotingOptions encapsula as opções de voto disponíveis.
type VotingOptions struct {
	List          []string
	DisplayString string
}

// Message representa um pacote UDP estruturado (JSON).
type Message struct {
	Type       string         `json:"type"`        // REGISTER, VOTE, BROADCAST, ACK, ERROR
	ClientID   string         `json:"client_id"`   // Identificador do cliente
	VoteOption string         `json:"vote,omitempty"`
	Message    string         `json:"message,omitempty"`
	VoteCounts map[string]int `json:"vote_counts,omitempty"`
	SeqNum     int            `json:"seq_num,omitempty"` // Número sequencial do broadcast
}

// BroadcastUpdate encapsula dados para broadcast assíncrono.
type BroadcastUpdate struct {
	VoteCounts map[string]int
	SeqNum     int
}
