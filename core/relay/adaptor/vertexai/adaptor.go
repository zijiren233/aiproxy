package vertexai

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/labring/aiproxy/core/relay/utils"
)

type Adaptor struct{}

var _ adaptor.Adaptor = (*Adaptor)(nil)

func (a *Adaptor) DefaultBaseURL() string {
	return ""
}

func (a *Adaptor) SupportMode(m mode.Mode) bool {
	return m == mode.ChatCompletions || m == mode.Anthropic || m == mode.Gemini
}

type Config struct {
	Region    string
	Key       string
	ProjectID string
	ADCJSON   string
}

func (a *Adaptor) ConvertRequest(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	request *http.Request,
) (adaptor.ConvertResult, error) {
	aa := GetAdaptor(meta.ActualModel)
	if aa == nil {
		return adaptor.ConvertResult{}, errors.New("adaptor not found")
	}

	result, err := aa.ConvertRequest(meta, store, request)
	if err != nil {
		return result, err
	}

	// Merge GetRequestURL logic
	var suffix string

	// For Gemini mode, get stream flag from URL
	var isStream bool
	if meta.Mode == mode.Gemini && c != nil {
		// Original URL path contains the action like :streamGenerateContent or :generateContent
		originalPath := c.Request.URL.Path
		isStream = strings.Contains(originalPath, ":stream")
	} else {
		isStream = meta.GetBool("stream")
	}

	if strings.HasPrefix(meta.ActualModel, "gemini") {
		if isStream {
			suffix = "streamGenerateContent?alt=sse"
		} else {
			suffix = "generateContent"
		}
	} else {
		if isStream {
			suffix = "streamRawPredict?alt=sse"
		} else {
			suffix = "rawPredict"
		}
	}

	config, err := getConfigFromKey(meta.Channel.Key)
	if err != nil {
		return result, err
	}

	publishers := "google"
	if strings.Contains(meta.ActualModel, "claude") {
		publishers = "anthropic"
	}

	var requestURL string

	result.Method = http.MethodPost

	if meta.Channel.BaseURL != "" {
		if config.ProjectID == "" || config.Region == "" {
			requestURL = fmt.Sprintf(
				"%s/v1/publishers/%s/models/%s:%s",
				meta.Channel.BaseURL,
				publishers,
				meta.ActualModel,
				suffix,
			)
		} else {
			requestURL = fmt.Sprintf(
				"%s/v1/projects/%s/locations/%s/publishers/%s/models/%s:%s",
				meta.Channel.BaseURL,
				config.ProjectID,
				config.Region,
				publishers,
				meta.ActualModel,
				suffix,
			)
		}
	} else {
		var requestDoamin string
		if config.Region == "" || config.Region == "global" {
			requestDoamin = "aiplatform.googleapis.com"
		} else {
			requestDoamin = config.Region + "-aiplatform.googleapis.com"
		}

		if config.ProjectID == "" || config.Region == "" {
			requestURL = fmt.Sprintf(
				"https://%s/v1/publishers/%s/models/%s:%s",
				requestDoamin,
				publishers,
				meta.ActualModel,
				suffix,
			)
		} else {
			requestURL = fmt.Sprintf(
				"https://%s/v1/projects/%s/locations/%s/publishers/%s/models/%s:%s",
				requestDoamin,
				config.ProjectID,
				config.Region,
				publishers,
				meta.ActualModel,
				suffix,
			)
		}
	}

	result.URL = requestURL

	return result, nil
}

func (a *Adaptor) DoResponse(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.UsageResult, adaptor.Error) {
	innerAdaptor := GetAdaptor(meta.ActualModel)
	if innerAdaptor == nil {
		return nil, relaymodel.WrapperOpenAIErrorWithMessage(
			meta.ActualModel+" adaptor not found",
			"adaptor_not_found",
			http.StatusInternalServerError,
		)
	}

	usage, err := innerAdaptor.DoResponse(meta, store, c, resp)
	if err != nil {
		return nil, err
	}

	return adaptor.NewSyncUsage(usage), nil
}

func (a *Adaptor) Metadata() adaptor.Metadata {
	return adaptor.Metadata{
		Readme:  "Claude support native Endpoint: /v1/messages\nGemini support",
		KeyHelp: "region|adcJSON or region|apikey or region|project_id|apikey",
		Models:  modelList,
	}
}

func (a *Adaptor) SetupRequestHeader(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	req *http.Request,
) error {
	innerAdaptor := GetAdaptor(meta.ActualModel)
	if innerAdaptor == nil {
		return relaymodel.WrapperOpenAIErrorWithMessage(
			meta.ActualModel+" adaptor not found",
			"adaptor_not_found",
			http.StatusInternalServerError,
		)
	}

	err := innerAdaptor.SetupRequestHeader(meta, store, c, req)
	if err != nil {
		return err
	}

	config, err := getConfigFromKey(meta.Channel.Key)
	if err != nil {
		return err
	}

	if config.Key != "" {
		req.Header.Set("X-Goog-Api-Key", config.Key)
		return nil
	}

	token, err := getToken(context.Background(), config.ADCJSON)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+token)

	return nil
}

func (a *Adaptor) DoRequest(
	meta *meta.Meta,
	_ adaptor.Store,
	_ *gin.Context,
	req *http.Request,
) (*http.Response, error) {
	return utils.DoRequest(req, meta.RequestTimeout)
}
