package jina

import (
	"io"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/model"
)

type ErrorResponse struct {
	Detail []struct {
		Loc  []string `json:"loc"`
		Msg  string   `json:"msg"`
		Type string   `json:"type"`
	} `json:"detail"`
}

func ErrorHanlder(resp *http.Response) *model.ErrorWithStatusCode {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return openai.ErrorWrapper(err, "read_response_body_failed", resp.StatusCode)
	}

	var jinaError ErrorResponse
	if err := sonic.Unmarshal(body, &jinaError); err != nil {
		return openai.ErrorWrapper(err, "unmarshal_response_body_failed", resp.StatusCode)
	}

	errorMessage := "unknown error"
	if len(jinaError.Detail) > 0 {
		errorMessage = jinaError.Detail[0].Msg
	}
	errorType := openai.ErrorTypeUpstream
	if len(jinaError.Detail) > 0 {
		errorType = jinaError.Detail[0].Type
	}

	return &model.ErrorWithStatusCode{
		Error: model.Error{
			Message: errorMessage,
			Type:    errorType,
			Code:    resp.StatusCode,
		},
		StatusCode: resp.StatusCode,
	}
}
