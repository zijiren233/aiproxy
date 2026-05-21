package siliconflow

import (
	"bytes"
	"net/http"
	"strconv"
	"time"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/common/image"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
)

type ImageRequest struct {
	Model             string `json:"model"`
	Prompt            string `json:"prompt"`
	NegativePrompt    string `json:"negative_prompt"`
	ImageSize         string `json:"image_size"`
	BatchSize         int    `json:"batch_size"`
	Seed              int64  `json:"seed"`
	NumInferenceSteps int    `json:"num_inference_steps"`
	GuidanceScale     int    `json:"guidance_scale"`
	PromptEnhancement bool   `json:"prompt_enhancement"`
}

type imageResponse struct {
	Images  []imageResponseImage `json:"images"`
	Timings map[string]any       `json:"timings,omitempty"`
	Seed    int64                `json:"seed,omitempty"`
}

type imageResponseImage struct {
	URL string `json:"url"`
}

func ConvertImageRequest(meta *meta.Meta, request *http.Request) (adaptor.ConvertResult, error) {
	var reqMap map[string]any

	err := common.UnmarshalRequestReusable(request, &reqMap)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	meta.Set(openai.MetaResponseFormat, reqMap["response_format"])

	reqMap["model"] = meta.ActualModel
	if _, ok := reqMap["n"]; ok {
		reqMap["batch_size"] = reqMap["n"]
		delete(reqMap, "n")
	}

	if _, ok := reqMap["steps"]; ok {
		reqMap["num_inference_steps"] = reqMap["steps"]
		delete(reqMap, "steps")
	}

	if _, ok := reqMap["scale"]; ok {
		reqMap["guidance_scale"] = reqMap["scale"]
		delete(reqMap, "scale")
	}

	if _, ok := reqMap["size"]; ok {
		reqMap["image_size"] = reqMap["size"]
		delete(reqMap, "size")
	}

	data, err := sonic.Marshal(&reqMap)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	return adaptor.ConvertResult{
		Header: http.Header{
			"Content-Type":   {"application/json"},
			"Content-Length": {strconv.Itoa(len(data))},
		},
		Body: bytes.NewReader(data),
	}, nil
}

func ImageHandler(
	meta *meta.Meta,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return adaptor.DoResponseResult{}, ErrorHandler(resp)
	}

	defer resp.Body.Close()

	log := common.GetLogger(c)

	var sfResponse imageResponse
	if err := common.UnmarshalResponse(resp, &sfResponse); err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIError(
			err,
			"unmarshal_response_body_failed",
			http.StatusInternalServerError,
		)
	}

	openaiResponse := relaymodel.ImageResponse{
		Created: time.Now().Unix(),
		Data:    make([]*relaymodel.ImageData, 0, len(sfResponse.Images)),
	}

	for _, img := range sfResponse.Images {
		openaiResponse.Data = append(openaiResponse.Data, &relaymodel.ImageData{
			URL: img.URL,
		})
	}

	var err error

	if meta.GetString(openai.MetaResponseFormat) == "b64_json" {
		for i := range openaiResponse.Data {
			data := openaiResponse.Data[i]
			if data.B64Json != "" || data.URL == "" {
				continue
			}

			_, data.B64Json, err = image.GetImageFromURL(c.Request.Context(), data.URL)
			if err != nil {
				log.Warnf(
					"convert siliconflow image url to b64_json failed, keep original url: %v",
					err,
				)

				continue
			}
		}
	}

	data, err := sonic.Marshal(openaiResponse)
	if err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIError(
			err,
			"marshal_response_body_failed",
			http.StatusInternalServerError,
		)
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(data)))

	_, err = c.Writer.Write(data)
	if err != nil {
		log.Warnf("write response body failed: %v", err)
	}

	usage := model.Usage{
		InputTokens:  meta.RequestUsage.InputTokens,
		OutputTokens: meta.RequestUsage.OutputTokens,
		TotalTokens:  meta.RequestUsage.InputTokens + meta.RequestUsage.OutputTokens,
	}

	return adaptor.DoResponseResult{Usage: usage}, nil
}
