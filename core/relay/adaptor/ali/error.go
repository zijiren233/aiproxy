package ali

import (
	"net/http"

	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
)

func ErrorHanlder(resp *http.Response) adaptor.Error {
	statusCode, openAIError := openai.GetError(resp)

	// {"error":{"code":"ServiceUnavailable","message":"<503> InternalError.Algo: An error occurred in model serving, error message is: [Too many requests. Your requests are being throttled due to system capacity limits. Please try again later.]","type":"ServiceUnavailable"}}
	if openAIError.Type == "ServiceUnavailable" {
		statusCode = http.StatusServiceUnavailable
		openAIError.Type = relaymodel.ErrorTypeUpstream
	}
	return relaymodel.NewOpenAIError(statusCode, openAIError)
}
