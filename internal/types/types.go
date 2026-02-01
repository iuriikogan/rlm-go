package types

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type UsageSummary struct {
	TotalCalls        int `json:"total_calls"`
	TotalInputTokens  int `json:"total_input_tokens"`
	TotalOutputTokens int `json:"total_output_tokens"`
}

type RLMChatCompletion struct {
	RootModel     string       `json:"root_model"`
	Prompt        interface{}  `json:"prompt"`
	Response      string       `json:"response"`
	UsageSummary  UsageSummary `json:"usage_summary"`
	ExecutionTime float64      `json:"execution_time"`
}

type REPLResult struct {
	Stdout        string              `json:"stdout"`
	Stderr        string              `json:"stderr"`
	ExecutionTime float64             `json:"execution_time"`
	RLMCalls      []RLMChatCompletion `json:"rlm_calls"`
}

type CodeBlock struct {
	Code   string     `json:"code"`
	Result REPLResult `json:"result"`
}

type RLMIteration struct {
	Prompt        []Message   `json:"prompt"`
	Response      string      `json:"response"`
	CodeBlocks    []CodeBlock `json:"code_blocks"`
	IterationTime float64     `json:"iteration_time"`
	FinalAnswer   string      `json:"final_answer,omitempty"`
}

type RLMMetadata struct {
	RootModel         string                 `json:"root_model"`
	MaxDepth          int                    `json:"max_depth"`
	MaxIterations     int                    `json:"max_iterations"`
	Backend           string                 `json:"backend"`
	BackendKwargs     map[string]interface{} `json:"backend_kwargs"`
	EnvironmentType   string                 `json:"environment_type"`
	EnvironmentKwargs map[string]interface{} `json:"environment_kwargs"`
}
