package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/middleware"
	"github.com/labring/aiproxy/relay/mode"

	// relay model used by swagger
	_ "github.com/labring/aiproxy/relay/model"
)

// @Summary		Completions
// @Description	Completions
// @Tags			relay
// @Produce		json
// @Security		ApiKeyAuth
// @Param			request	body		model.GeneralOpenAIRequest	true	"Request"
// @Success		200		{object}	model.TextResponse
// @Router			/v1/completions [post]
func Completions() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.NewDistribute(mode.Completions),
		NewRelay(mode.Completions),
	}
}

// @Summary		ChatCompletions
// @Description	ChatCompletions
// @Tags			relay
// @Produce		json
// @Security		ApiKeyAuth
// @Param			request	body		model.GeneralOpenAIRequest	true	"Request"
// @Success		200		{object}	model.TextResponse			|		model.ChatCompletionsStreamResponse
// @Router			/v1/chat/completions [post]
func ChatCompletions() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.NewDistribute(mode.ChatCompletions),
		NewRelay(mode.ChatCompletions),
	}
}

// @Summary		Embeddings
// @Description	Embeddings
// @Tags			relay
// @Produce		json
// @Security		ApiKeyAuth
// @Param			request	body		model.EmbeddingRequest	true	"Request"
// @Success		200		{object}	model.EmbeddingResponse
// @Router			/v1/embeddings [post]
func Embeddings() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.NewDistribute(mode.Embeddings),
		NewRelay(mode.Embeddings),
	}
}

func Edits() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.NewDistribute(mode.Edits),
		NewRelay(mode.Edits),
	}
}

// @Summary		ImagesGenerations
// @Description	ImagesGenerations
// @Tags			relay
// @Produce		json
// @Security		ApiKeyAuth
// @Param			request	body		model.ImageRequest	true	"Request"
// @Success		200		{object}	model.ImageResponse
// @Router			/v1/images/generations [post]
func ImagesGenerations() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.NewDistribute(mode.ImagesGenerations),
		NewRelay(mode.ImagesGenerations),
	}
}

// @Summary		AudioSpeech
// @Description	AudioSpeech
// @Tags			relay
// @Produce		json
// @Security		ApiKeyAuth
// @Param			request	body	model.TextToSpeechRequest	true	"Request"
// @Success		200		{file}	file						"audio binary"
// @Router			/v1/audio/speech [post]
func AudioSpeech() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.NewDistribute(mode.AudioSpeech),
		NewRelay(mode.AudioSpeech),
	}
}

// @Summary		AudioTranscription
// @Description	AudioTranscription
// @Tags			relay
// @Produce		json
// @Security		ApiKeyAuth
// @Param			model	formData	string	true	"Model"
// @Param			file	formData	file	true	"File"
// @Success		200		{object}	model.SttJSONResponse
// @Router			/v1/audio/transcription [post]
func AudioTranscription() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.NewDistribute(mode.AudioTranscription),
		NewRelay(mode.AudioTranscription),
	}
}

// @Summary		AudioTranslation
// @Description	AudioTranslation
// @Tags			relay
// @Produce		json
// @Security		ApiKeyAuth
// @Param			model	formData	string	true	"Model"
// @Param			file	formData	file	true	"File"
// @Success		200		{object}	model.SttJSONResponse
// @Router			/v1/audio/translation [post]
func AudioTranslation() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.NewDistribute(mode.AudioTranslation),
		NewRelay(mode.AudioTranslation),
	}
}

func Moderations() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.NewDistribute(mode.Moderations),
		NewRelay(mode.Moderations),
	}
}

// @Summary		Rerank
// @Description	Rerank
// @Tags			relay
// @Produce		json
// @Security		ApiKeyAuth
// @Param			request	body		model.RerankRequest	true	"Request"
// @Success		200		{object}	model.RerankResponse
// @Router			/v1/rerank [post]
func Rerank() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.NewDistribute(mode.Rerank),
		NewRelay(mode.Rerank),
	}
}

// @Summary		ParsePdf
// @Description	ParsePdf
// @Tags			relay
// @Produce		json
// @Security		ApiKeyAuth
// @Param			model	formData	string	true	"Model"
// @Param			file	formData	file	true	"File"
// @Success		200		{object}	model.ParsePdfResponse
// @Router			/v1/parse-pdf [post]
func ParsePdf() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		middleware.NewDistribute(mode.ParsePdf),
		NewRelay(mode.ParsePdf),
	}
}
