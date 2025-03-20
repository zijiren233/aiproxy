package model

type SttJSONResponse struct {
	Text string `json:"text,omitempty"`
}

type SttVerboseJSONResponse struct {
	Task     string     `json:"task,omitempty"`
	Language string     `json:"language,omitempty"`
	Text     string     `json:"text,omitempty"`
	Segments []*Segment `json:"segments,omitempty"`
	Duration float64    `json:"duration,omitempty"`
}

type Segment struct {
	Text             string  `json:"text"`
	Tokens           []int   `json:"tokens"`
	ID               int     `json:"id"`
	Seek             int     `json:"seek"`
	Start            float64 `json:"start"`
	End              float64 `json:"end"`
	Temperature      float64 `json:"temperature"`
	AvgLogprob       float64 `json:"avg_logprob"`
	CompressionRatio float64 `json:"compression_ratio"`
	NoSpeechProb     float64 `json:"no_speech_prob"`
}
