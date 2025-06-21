package minimax

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/labring/aiproxy/core/relay/adaptor"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
)

type BaseResp struct {
	StatusMsg  string `json:"status_msg"`
	StatusCode int    `json:"status_code"`
}

type ErrorResponse struct {
	BaseResp *BaseResp `json:"base_resp"`
}

func TryErrorHanlder(resp *http.Response) adaptor.Error {
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return relaymodel.NewOpenAIError(resp.StatusCode, relaymodel.OpenAIError{
			Message: err.Error(),
			Type:    relaymodel.ErrorTypeUpstream,
			Code:    relaymodel.ErrorCodeBadResponse,
		})
	}

	var result ErrorResponse
	if err := sonic.Unmarshal(respBody, &result); err != nil {
		return relaymodel.WrapperOpenAIError(
			err,
			"TTS_ERROR",
			http.StatusInternalServerError,
		)
	}
	if result.BaseResp != nil && result.BaseResp.StatusCode != 0 {
		statusCode := http.StatusInternalServerError
		switch {
		case result.BaseResp.StatusCode == 1008,
			strings.Contains(result.BaseResp.StatusMsg, "insufficient balance"):
			statusCode = http.StatusPaymentRequired
		case result.BaseResp.StatusCode == 1001:
			statusCode = http.StatusRequestTimeout
		case result.BaseResp.StatusCode == 1004:
			statusCode = http.StatusForbidden
		case result.BaseResp.StatusCode == 1026,
			result.BaseResp.StatusCode == 1027,
			result.BaseResp.StatusCode == 2013:
			statusCode = http.StatusBadRequest
		case result.BaseResp.StatusCode == 1039:
			statusCode = http.StatusTooManyRequests
		}
		return relaymodel.WrapperOpenAIErrorWithMessage(
			result.BaseResp.StatusMsg,
			strconv.Itoa(result.BaseResp.StatusCode),
			statusCode,
			relaymodel.ErrorTypeUpstream,
		)
	}

	resp.Body = io.NopCloser(bytes.NewReader(respBody))

	return nil
}
