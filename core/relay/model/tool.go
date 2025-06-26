package model

type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

type Function struct {
	Parameters  any    `json:"parameters,omitempty"`
	Arguments   string `json:"arguments,omitempty"`
	Description string `json:"description,omitempty"`
	Name        string `json:"name,omitempty"`
}

type ToolCall struct {
	Index    int      `json:"index"`
	ID       string   `json:"id"`
	Type     string   `json:"type"`
	Function Function `json:"function"`
}
