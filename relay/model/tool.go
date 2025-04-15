package model

type Tool struct {
	ID       string   `json:"id,omitempty"`
	Type     string   `json:"type,omitempty"`
	Function Function `json:"function"`
}

type Function struct {
	Parameters  any    `json:"parameters,omitempty"`
	Arguments   string `json:"arguments,omitempty"`
	Description string `json:"description,omitempty"`
	Name        string `json:"name,omitempty"`
}
