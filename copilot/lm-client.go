package copilot

import "github.com/go-skynet/go-llama.cpp"

type LM struct {
	model *llama.LLama
}
type LMOptions struct {
	ModelPath string
	Threads   int
	Grammar   string
}
