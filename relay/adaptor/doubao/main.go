package doubao

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/labring/aiproxy/model"
	"github.com/labring/aiproxy/relay/adaptor"
	"github.com/labring/aiproxy/relay/adaptor/openai"
	"github.com/labring/aiproxy/relay/meta"
	"github.com/labring/aiproxy/relay/mode"
	relaymodel "github.com/labring/aiproxy/relay/model"
)

func GetRequestURL(meta *meta.Meta) (string, error) {
	u := meta.Channel.BaseURL
	switch meta.Mode {
	case mode.ChatCompletions:
		if strings.HasPrefix(meta.ActualModel, "bot-") {
			return u + "/api/v3/bots/chat/completions", nil
		}
		return u + "/api/v3/chat/completions", nil
	case mode.Embeddings:
		return u + "/api/v3/embeddings", nil
	default:
		return "", fmt.Errorf("unsupported relay mode %d for doubao", meta.Mode)
	}
}

type Adaptor struct {
	openai.Adaptor
}

const baseURL = "https://ark.cn-beijing.volces.com"

func (a *Adaptor) GetBaseURL() string {
	return baseURL
}

func (a *Adaptor) GetModelList() []*model.ModelConfig {
	return ModelList
}

func (a *Adaptor) GetRequestURL(meta *meta.Meta) (string, error) {
	return GetRequestURL(meta)
}

func (a *Adaptor) ConvertRequest(meta *meta.Meta, req *http.Request) (string, http.Header, io.Reader, error) {
	method, header, body, err := a.Adaptor.ConvertRequest(meta, req)
	if err != nil {
		return "", nil, nil, err
	}
	if meta.Mode != mode.ChatCompletions || meta.OriginModel != "deepseek-reasoner" {
		return method, header, body, nil
	}

	m := make(map[string]any)
	err = sonic.ConfigDefault.NewDecoder(body).Decode(&m)
	if err != nil {
		return "", nil, nil, err
	}
	messages, _ := m["messages"].([]any)
	if len(messages) == 0 {
		return "", nil, nil, errors.New("messages is empty")
	}
	sysMessage := relaymodel.Message{
		Role:    "system",
		Content: "回答前，都先用 <think></think> 输出你的思考过程。",
	}
	messages = append([]any{sysMessage}, messages...)
	m["messages"] = messages
	newBody, err := sonic.Marshal(m)
	if err != nil {
		return "", nil, nil, err
	}

	return method, header, bytes.NewReader(newBody), nil
}

func (a *Adaptor) GetChannelName() string {
	return "doubao"
}

func (a *Adaptor) GetBalance(_ *model.Channel) (float64, error) {
	return 0, adaptor.ErrGetBalanceNotImplemented
}
