package model

type EmbeddingRequest struct {
	Input          string `json:"input"`
	Model          string `json:"model"`
	EncodingFormat string `json:"encoding_format"`
	Dimensions     int    `json:"dimensions"`
}

type EmbeddingResponseItem struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

type EmbeddingResponse struct {
	Object string                   `json:"object"`
	Model  string                   `json:"model"`
	Data   []*EmbeddingResponseItem `json:"data"`
	Usage  `json:"usage"`
}
