package openai

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common/conv"
	"github.com/labring/aiproxy/core/relay/model"
)

const (
	nn   = "\n\n"
	data = "data: "
	DONE = "[DONE]"
)

var (
	nnBytes   = conv.StringToBytes(nn)
	dataBytes = conv.StringToBytes(data)
)

type OpenAISSE struct {
	Data string
}

func (r *OpenAISSE) Render(w http.ResponseWriter) error {
	r.WriteContentType(w)

	for _, bytes := range [][]byte{
		dataBytes,
		conv.StringToBytes(r.Data),
		nnBytes,
	} {
		// nosemgrep:
		// go.lang.security.audit.xss.no-direct-write-to-responsewriter.no-direct-write-to-responsewriter
		if _, err := w.Write(bytes); err != nil {
			return err
		}
	}
	return nil
}

func (r *OpenAISSE) WriteContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("X-Accel-Buffering", "no")
}

func StringData(c *gin.Context, str string) {
	if len(c.Errors) > 0 {
		return
	}
	if c.IsAborted() {
		return
	}
	c.Render(-1, &OpenAISSE{Data: str})
	c.Writer.Flush()
}

func ObjectData(c *gin.Context, object any) error {
	if len(c.Errors) > 0 {
		return c.Errors.Last()
	}
	if c.IsAborted() {
		return errors.New("context aborted")
	}
	jsonData, err := sonic.Marshal(object)
	if err != nil {
		return fmt.Errorf("error marshalling object: %w", err)
	}
	c.Render(-1, &OpenAISSE{Data: conv.BytesToString(jsonData)})
	c.Writer.Flush()
	return nil
}

func Done(c *gin.Context) {
	StringData(c, DONE)
}

type OpenAITTSSSE struct {
	Audio string // base64 encode audio data
	Usage *model.TextToSpeechUsage
}

func (r *OpenAITTSSSE) Render(w http.ResponseWriter) error {
	r.WriteContentType(w)

	payload := model.TextToSpeechSSEResponse{
		Audio: r.Audio,
		Usage: r.Usage,
	}
	if r.Usage != nil {
		payload.Type = model.TextToSpeechSSEResponseTypeDone
	} else {
		payload.Type = model.TextToSpeechSSEResponseTypeDelta
	}

	jsonData, err := sonic.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshalling object: %w", err)
	}

	for _, bytes := range [][]byte{
		dataBytes,
		jsonData,
		nnBytes,
	} {
		// nosemgrep:
		// go.lang.security.audit.xss.no-direct-write-to-responsewriter.no-direct-write-to-responsewriter
		if _, err := w.Write(bytes); err != nil {
			return err
		}
	}
	return nil
}

func (r *OpenAITTSSSE) WriteContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("X-Accel-Buffering", "no")
}

func AudioData(c *gin.Context, audio string) {
	if len(c.Errors) > 0 {
		return
	}
	if c.IsAborted() {
		return
	}
	c.Render(-1, &OpenAITTSSSE{Audio: audio})
	c.Writer.Flush()
}

type AudioDataWriter struct {
	c *gin.Context
}

func NewAudioDataWriter(c *gin.Context) *AudioDataWriter {
	return &AudioDataWriter{c: c}
}

func (w *AudioDataWriter) Write(p []byte) (n int, err error) {
	AudioData(w.c, base64.StdEncoding.EncodeToString(p))
	return len(p), nil
}

func AudioDone(c *gin.Context, usage model.TextToSpeechUsage) {
	if len(c.Errors) > 0 {
		return
	}
	if c.IsAborted() {
		return
	}
	c.Render(-1, &OpenAITTSSSE{Usage: &usage})
	c.Writer.Flush()
}
