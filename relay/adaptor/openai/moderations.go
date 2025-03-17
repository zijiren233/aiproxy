package openai

import (
	"io"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/middleware"
	"github.com/labring/aiproxy/relay/meta"
	"github.com/labring/aiproxy/relay/model"
)

func ModerationsHandler(meta *meta.Meta, c *gin.Context, resp *http.Response) (*model.Usage, *model.ErrorWithStatusCode) {
	if resp.StatusCode != http.StatusOK {
		return nil, ErrorHanlder(resp)
	}

	defer resp.Body.Close()

	log := middleware.GetLogger(c)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, ErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}

	node, err := sonic.Get(body)
	if err != nil {
		return nil, ErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError)
	}

	if _, err := node.Set("model", ast.NewString(meta.OriginModel)); err != nil {
		return nil, ErrorWrapper(err, "set_model_failed", http.StatusInternalServerError)
	}

	newData, err := node.MarshalJSON()
	if err != nil {
		return nil, ErrorWrapper(err, "marshal_response_body_failed", http.StatusInternalServerError)
	}

	usage := &model.Usage{
		PromptTokens: meta.InputTokens,
		TotalTokens:  meta.InputTokens,
	}

	_, err = c.Writer.Write(newData)
	if err != nil {
		log.Warnf("write response body failed: %v", err)
	}
	return usage, nil
}
