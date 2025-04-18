package jina

import (
	"io"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/model"
)

type Detail struct {
	Loc  []string `json:"loc"`
	Msg  string   `json:"msg"`
	Type string   `json:"type"`
}

func ErrorHanlder(resp *http.Response) *model.ErrorWithStatusCode {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return openai.ErrorWrapper(err, "read_response_body_failed", resp.StatusCode)
	}

	detailValue, err := sonic.Get(body, "detail")
	if err != nil {
		return openai.ErrorWrapper(err, "unmarshal_response_body_failed", resp.StatusCode)
	}

	errorMessage := "unknown error"
	errorType := openai.ErrorTypeUpstream

	if detailStr, err := detailValue.String(); err == nil {
		errorMessage = detailStr
	} else {
		var details []Detail
		detailsData, _ := detailValue.Raw()
		if err := sonic.Unmarshal([]byte(detailsData), &details); err == nil && len(details) > 0 {
			errorMessage = details[0].Msg
			if details[0].Type != "" {
				errorType = details[0].Type
			}
		}
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
