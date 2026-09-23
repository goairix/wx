package work

// Config configures an enterprise WeChat client.
type Config struct {
	CorpID         string
	CorpSecret     string
	Token          string
	EncodingAESKey string
	AgentID        int64
}
