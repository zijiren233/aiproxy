package baidu

import (
	"io"
	"net/http"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/middleware"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
)

type ImageData struct {
	B64Image string `json:"b64_image"`
}

type ImageResponse struct {
	*Error
	ID      string       `json:"id"`
	Data    []*ImageData `json:"data"`
	Created int64        `json:"created"`
}

func ImageHandler(_ *meta.Meta, c *gin.Context, resp *http.Response) (*model.Usage, *relaymodel.ErrorWithStatusCode) {
	defer resp.Body.Close()

	log := middleware.GetLogger(c)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}
	var imageResponse ImageResponse
	err = sonic.Unmarshal(body, &imageResponse)
	if err != nil {
		return nil, openai.ErrorWrapper(err, "unmarshal_response_body_failed", http.StatusInternalServerError)
	}

	usage := &model.Usage{
		InputTokens: model.ZeroNullInt64(len(imageResponse.Data)),
		TotalTokens: model.ZeroNullInt64(len(imageResponse.Data)),
	}

	if imageResponse.Error != nil && imageResponse.Error.ErrorMsg != "" {
		return usage, ErrorHandler(imageResponse.Error)
	}

	openaiResponse := ToOpenAIImageResponse(&imageResponse)
	data, err := sonic.Marshal(openaiResponse)
	if err != nil {
		return usage, openai.ErrorWrapper(err, "marshal_response_body_failed", http.StatusInternalServerError)
	}
	_, err = c.Writer.Write(data)
	if err != nil {
		log.Warnf("write response body failed: %v", err)
	}
	return usage, nil
}

func ToOpenAIImageResponse(imageResponse *ImageResponse) *relaymodel.ImageResponse {
	response := &relaymodel.ImageResponse{
		Created: imageResponse.Created,
	}
	for _, data := range imageResponse.Data {
		response.Data = append(response.Data, &relaymodel.ImageData{
			B64Json: data.B64Image,
		})
	}
	return response
}
