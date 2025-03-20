package xai

import (
	"io"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/labring/aiproxy/common/conv"
	"github.com/labring/aiproxy/relay/adaptor/openai"
	model "github.com/labring/aiproxy/relay/model"
)

type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func ErrorHandler(resp *http.Response) *model.ErrorWithStatusCode {
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return openai.ErrorWrapperWithMessage("read response body error: "+err.Error(), nil, http.StatusInternalServerError)
	}

	var er errorResponse
	err = sonic.Unmarshal(data, &er)
	if err != nil {
		return openai.ErrorWrapperWithMessage(conv.BytesToString(data), nil, http.StatusInternalServerError)
	}
	return openai.ErrorWrapperWithMessage(er.Error, er.Code, resp.StatusCode)
}
