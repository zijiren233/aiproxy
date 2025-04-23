package openai

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/labring/aiproxy/core/relay/utils"
)

var _ adaptor.Adaptor = (*Adaptor)(nil)

type Adaptor struct{}

const baseURL = "https://api.openai.com/v1"

func (a *Adaptor) GetBaseURL() string {
	return baseURL
}

func (a *Adaptor) GetRequestURL(meta *meta.Meta) (string, error) {
	u := meta.Channel.BaseURL

	var path string
	switch meta.Mode {
	case mode.ChatCompletions:
		path = "/chat/completions"
	case mode.Completions:
		path = "/completions"
	case mode.Embeddings:
		path = "/embeddings"
	case mode.Moderations:
		path = "/moderations"
	case mode.ImagesGenerations:
		path = "/images/generations"
	case mode.Edits:
		path = "/edits"
	case mode.AudioSpeech:
		path = "/audio/speech"
	case mode.AudioTranscription:
		path = "/audio/transcriptions"
	case mode.AudioTranslation:
		path = "/audio/translations"
	case mode.Rerank:
		path = "/rerank"
	default:
		return "", fmt.Errorf("unsupported mode: %s", meta.Mode)
	}

	return u + path, nil
}

func (a *Adaptor) SetupRequestHeader(meta *meta.Meta, _ *gin.Context, req *http.Request) error {
	req.Header.Set("Authorization", "Bearer "+meta.Channel.Key)
	return nil
}

func (a *Adaptor) ConvertRequest(meta *meta.Meta, req *http.Request) (string, http.Header, io.Reader, error) {
	return ConvertRequest(meta, req)
}

func ConvertRequest(meta *meta.Meta, req *http.Request) (string, http.Header, io.Reader, error) {
	if req == nil {
		return "", nil, nil, errors.New("request is nil")
	}
	switch meta.Mode {
	case mode.Moderations:
		return ConvertEmbeddingsRequest(meta, req, true)
	case mode.Embeddings, mode.Completions:
		return ConvertEmbeddingsRequest(meta, req, false)
	case mode.ChatCompletions:
		return ConvertTextRequest(meta, req, false)
	case mode.ImagesGenerations:
		return ConvertImageRequest(meta, req)
	case mode.AudioTranscription, mode.AudioTranslation:
		return ConvertSTTRequest(meta, req)
	case mode.AudioSpeech:
		return ConvertTTSRequest(meta, req, "")
	case mode.Rerank:
		return ConvertRerankRequest(meta, req)
	default:
		return "", nil, nil, fmt.Errorf("unsupported mode: %s", meta.Mode)
	}
}

func DoResponse(meta *meta.Meta, c *gin.Context, resp *http.Response) (usage *model.Usage, err *relaymodel.ErrorWithStatusCode) {
	switch meta.Mode {
	case mode.ImagesGenerations:
		usage, err = ImageHandler(meta, c, resp)
	case mode.AudioTranscription, mode.AudioTranslation:
		usage, err = STTHandler(meta, c, resp)
	case mode.AudioSpeech:
		usage, err = TTSHandler(meta, c, resp)
	case mode.Rerank:
		usage, err = RerankHandler(meta, c, resp)
	case mode.Moderations:
		usage, err = ModerationsHandler(meta, c, resp)
	case mode.Embeddings, mode.Completions:
		fallthrough
	case mode.ChatCompletions:
		if utils.IsStreamResponse(resp) {
			usage, err = StreamHandler(meta, c, resp, nil)
		} else {
			usage, err = Handler(meta, c, resp, nil)
		}
	default:
		return nil, ErrorWrapperWithMessage(fmt.Sprintf("unsupported mode: %s", meta.Mode), "unsupported_mode", http.StatusBadRequest)
	}
	return usage, err
}

func ConvertTextRequest(meta *meta.Meta, req *http.Request, doNotPatchStreamOptionsIncludeUsage bool) (string, http.Header, io.Reader, error) {
	reqMap := make(map[string]any)
	err := common.UnmarshalBodyReusable(req, &reqMap)
	if err != nil {
		return "", nil, nil, err
	}

	if !doNotPatchStreamOptionsIncludeUsage {
		if err := patchStreamOptions(reqMap); err != nil {
			return "", nil, nil, err
		}
	}

	reqMap["model"] = meta.ActualModel
	jsonData, err := sonic.Marshal(reqMap)
	if err != nil {
		return "", nil, nil, err
	}
	return http.MethodPost, nil, bytes.NewReader(jsonData), nil
}

func patchStreamOptions(reqMap map[string]any) error {
	stream, ok := reqMap["stream"]
	if !ok {
		return nil
	}

	streamBool, ok := stream.(bool)
	if !ok {
		return errors.New("stream is not a boolean")
	}

	if !streamBool {
		return nil
	}

	streamOptions, ok := reqMap["stream_options"].(map[string]any)
	if !ok {
		if reqMap["stream_options"] != nil {
			return errors.New("stream_options is not a map")
		}
		reqMap["stream_options"] = map[string]any{
			"include_usage": true,
		}
		return nil
	}

	streamOptions["include_usage"] = true
	return nil
}

const MetaResponseFormat = "response_format"

func (a *Adaptor) DoRequest(_ *meta.Meta, _ *gin.Context, req *http.Request) (*http.Response, error) {
	return utils.DoRequest(req)
}

func (a *Adaptor) DoResponse(meta *meta.Meta, c *gin.Context, resp *http.Response) (usage *model.Usage, err *relaymodel.ErrorWithStatusCode) {
	return DoResponse(meta, c, resp)
}

func (a *Adaptor) GetModelList() []*model.ModelConfig {
	return ModelList
}

func (a *Adaptor) GetChannelName() string {
	return "openai"
}
