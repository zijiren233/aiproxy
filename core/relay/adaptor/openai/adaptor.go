package openai

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
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

// /v1/video/generations/jobs/{job_id}
func (a *Adaptor) getJobID(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

// /v1/video/generations/{generation_id}/content/video
func (a *Adaptor) getGenerationID(path string) string {
	if strings.HasSuffix(path, "/content/video") {
		parts := strings.Split(path, "/")
		if len(parts) == 0 {
			return ""
		}
		return parts[len(parts)-2]
	}
	return ""
}

func (a *Adaptor) GetRequestURL(meta *meta.Meta, _ adaptor.Store) (string, error) {
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
	case mode.ImagesEdits:
		path = "/images/edits"
	case mode.AudioSpeech:
		path = "/audio/speech"
	case mode.AudioTranscription:
		path = "/audio/transcriptions"
	case mode.AudioTranslation:
		path = "/audio/translations"
	case mode.Rerank:
		path = "/rerank"
	case mode.VideoGenerationsJobs:
		path = "/video/generations/jobs"
	case mode.VideoGenerationsGetJobs:
		path = "/video/generations/jobs/" + a.getJobID(meta.Endpoint)
	case mode.VideoGenerationsContent:
		path = fmt.Sprintf("/video/generations/%s/content/video", a.getGenerationID(meta.Endpoint))
	default:
		return "", fmt.Errorf("unsupported mode: %s", meta.Mode)
	}

	return u + path, nil
}

func (a *Adaptor) SetupRequestHeader(
	meta *meta.Meta,
	_ adaptor.Store,
	_ *gin.Context,
	req *http.Request,
) error {
	req.Header.Set("Authorization", "Bearer "+meta.Channel.Key)
	return nil
}

func (a *Adaptor) ConvertRequest(
	meta *meta.Meta,
	store adaptor.Store,
	req *http.Request,
) (*adaptor.ConvertRequestResult, error) {
	return ConvertRequest(meta, store, req)
}

func ConvertRequest(
	meta *meta.Meta,
	_ adaptor.Store,
	req *http.Request,
) (*adaptor.ConvertRequestResult, error) {
	if req == nil {
		return nil, errors.New("request is nil")
	}
	switch meta.Mode {
	case mode.Moderations:
		return ConvertEmbeddingsRequest(meta, req, true)
	case mode.Embeddings, mode.Completions:
		return ConvertEmbeddingsRequest(meta, req, false)
	case mode.ChatCompletions:
		return ConvertTextRequest(meta, req, false)
	case mode.ImagesGenerations:
		return ConvertImagesRequest(meta, req)
	case mode.ImagesEdits:
		return ConvertImagesEditsRequest(meta, req)
	case mode.AudioTranscription, mode.AudioTranslation:
		return ConvertSTTRequest(meta, req)
	case mode.AudioSpeech:
		return ConvertTTSRequest(meta, req, "")
	case mode.Rerank:
		return ConvertRerankRequest(meta, req)
	case mode.VideoGenerationsJobs:
		return ConvertVideoRequest(meta, req)
	case mode.VideoGenerationsGetJobs:
		return ConvertVideoGetJobsRequest(meta, req)
	case mode.VideoGenerationsContent:
		return ConvertVideoGetJobsContentRequest(meta, req)
	default:
		return nil, fmt.Errorf("unsupported mode: %s", meta.Mode)
	}
}

func DoResponse(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (usage *model.Usage, err adaptor.Error) {
	switch meta.Mode {
	case mode.ImagesGenerations, mode.ImagesEdits:
		usage, err = ImagesHandler(meta, c, resp)
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
	case mode.VideoGenerationsJobs:
		usage, err = VideoHandler(meta, store, c, resp)
	case mode.VideoGenerationsGetJobs:
		usage, err = VideoGetJobsHandler(meta, store, c, resp)
	case mode.VideoGenerationsContent:
		usage, err = VideoGetJobsContentHandler(meta, store, c, resp)
	default:
		return nil, relaymodel.WrapperOpenAIErrorWithMessage(
			fmt.Sprintf("unsupported mode: %s", meta.Mode),
			"unsupported_mode",
			http.StatusBadRequest,
		)
	}
	return usage, err
}

const MetaResponseFormat = "response_format"

func (a *Adaptor) DoRequest(
	_ *meta.Meta,
	_ adaptor.Store,
	_ *gin.Context,
	req *http.Request,
) (*http.Response, error) {
	return utils.DoRequest(req)
}

func (a *Adaptor) DoResponse(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (usage *model.Usage, err adaptor.Error) {
	return DoResponse(meta, store, c, resp)
}

func (a *Adaptor) GetModelList() []model.ModelConfig {
	return ModelList
}
