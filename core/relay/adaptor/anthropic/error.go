package anthropic

import (
	"net/http"
	"strings"

	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/model"
)

// status 400 {"type":"error","error":{"type":"invalid_request_error","message":"Your credit balance is too low to access the Anthropic API. Please go to Plans & Billing to upgrade or purchase credits."}}
// status 529 {Message:Overloaded Type:overloaded_error Param:}
func OpenAIErrorHandler(resp *http.Response) *model.ErrorWithStatusCode {
	err := openai.ErrorHanlder(resp)
	if strings.Contains(err.Error.Message, "balance is too low") {
		err.StatusCode = http.StatusPaymentRequired
		return err
	}
	if strings.Contains(err.Error.Message, "Overloaded") {
		err.StatusCode = http.StatusTooManyRequests
		return err
	}
	return err
}

func OpenAIErrorHandlerWithBody(statusCode int, respBody []byte) *model.ErrorWithStatusCode {
	err := openai.ErrorHanlderWithBody(statusCode, respBody)
	if strings.Contains(err.Error.Message, "balance is too low") {
		err.StatusCode = http.StatusPaymentRequired
		return err
	}
	if strings.Contains(err.Error.Message, "Overloaded") {
		err.StatusCode = http.StatusTooManyRequests
		return err
	}
	return err
}
