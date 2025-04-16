package model

type TextToSpeechRequest struct {
	Model          string  `binding:"required"     json:"model"`
	Input          string  `binding:"required"     json:"input"`
	Voice          string  `binding:"required"     json:"voice"`
	ResponseFormat string  `json:"response_format"`
	Speed          float64 `json:"speed"`
}
