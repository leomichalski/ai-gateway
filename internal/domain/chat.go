package domain

type ChatResult struct {
	Answer     string
	TokensUsed int32
	Provider   string
}
