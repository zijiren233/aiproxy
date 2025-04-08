package vertexai

import (
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/model"
	"github.com/labring/aiproxy/relay/adaptor/gemini"
	vertexclaude "github.com/labring/aiproxy/relay/adaptor/vertexai/claude"
	vertexgemini "github.com/labring/aiproxy/relay/adaptor/vertexai/gemini"
	"github.com/labring/aiproxy/relay/meta"
	relaymodel "github.com/labring/aiproxy/relay/model"
)

type ModelType int

const (
	VerterAIClaude ModelType = iota + 1
	VerterAIGemini
)

var modelList = []*model.ModelConfig{}

func init() {
	modelList = append(modelList, vertexclaude.ModelList...)

	modelList = append(modelList, gemini.ModelList...)
}

type innerAIAdapter interface {
	ConvertRequest(meta *meta.Meta, request *http.Request) (string, http.Header, io.Reader, error)
	DoResponse(meta *meta.Meta, c *gin.Context, resp *http.Response) (usage *model.Usage, err *relaymodel.ErrorWithStatusCode)
}

func GetAdaptor(model string) innerAIAdapter {
	switch {
	case strings.Contains(model, "claude"):
		return &vertexclaude.Adaptor{}
	case strings.Contains(model, "gemini"):
		return &vertexgemini.Adaptor{}
	default:
		return nil
	}
}
