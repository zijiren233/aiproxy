package openai

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/common/conv"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/meta"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/labring/aiproxy/core/relay/utils"
)

func ConvertSTTRequest(
	meta *meta.Meta,
	request *http.Request,
) (adaptor.ConvertResult, error) {
	err := request.ParseMultipartForm(1024 * 1024 * 4)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	multipartBody := &bytes.Buffer{}
	multipartWriter := multipart.NewWriter(multipartBody)

	for key, values := range request.MultipartForm.Value {
		if len(values) == 0 {
			continue
		}

		value := values[0]

		if key == "model" {
			err = multipartWriter.WriteField(key, meta.ActualModel)
			if err != nil {
				return adaptor.ConvertResult{}, err
			}

			continue
		}

		if key == "response_format" {
			meta.Set(MetaResponseFormat, value)
			continue
		}

		err = multipartWriter.WriteField(key, value)
		if err != nil {
			return adaptor.ConvertResult{}, err
		}
	}

	for key, files := range request.MultipartForm.File {
		if len(files) == 0 {
			continue
		}

		fileHeader := files[0]

		file, err := fileHeader.Open()
		if err != nil {
			return adaptor.ConvertResult{}, err
		}

		w, err := multipartWriter.CreateFormFile(key, fileHeader.Filename)
		if err != nil {
			file.Close()
			return adaptor.ConvertResult{}, err
		}

		_, err = io.Copy(w, file)
		file.Close()

		if err != nil {
			return adaptor.ConvertResult{}, err
		}
	}

	multipartWriter.Close()
	ContentType := multipartWriter.FormDataContentType()

	return adaptor.ConvertResult{
		Header: http.Header{
			"Content-Type": {ContentType},
		},
		Body: multipartBody,
	}, nil
}

func STTHandler(
	meta *meta.Meta,
	c *gin.Context,
	resp *http.Response,
) (model.Usage, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return model.Usage{}, ErrorHanlder(resp)
	}

	if utils.IsStreamResponse(resp) {
		return sttStreamHandler(meta, c, resp)
	}

	defer resp.Body.Close()

	log := common.GetLogger(c)

	responseFormat := meta.GetString(MetaResponseFormat)

	responseBody, err := common.GetResponseBody(resp)
	if err != nil {
		return model.Usage{}, relaymodel.WrapperOpenAIError(
			err,
			"read_response_body_failed",
			http.StatusInternalServerError,
		)
	}

	var text string
	switch responseFormat {
	case "text":
		text = getTextFromText(responseBody)
	case "srt":
		text, err = getTextFromSRT(responseBody)
	case "verbose_json":
		text, err = getTextFromVerboseJSON(responseBody)
	case "vtt":
		text, err = getTextFromVTT(responseBody)
	case "json":
		fallthrough
	default:
		text, err = getTextFromJSON(responseBody)
	}

	if err != nil {
		return model.Usage{}, relaymodel.WrapperOpenAIError(
			err,
			"get_text_from_body_err",
			http.StatusInternalServerError,
		)
	}

	outputTokens := CountTokenText(text, meta.ActualModel)
	usage := relaymodel.SttUsage{
		Type:         relaymodel.SttUsageTypeTokens,
		Seconds:      int64(meta.RequestUsage.AudioInputTokens),
		InputTokens:  int64(meta.RequestUsage.InputTokens),
		OutputTokens: outputTokens,
		InputTokenDetails: &relaymodel.SttUsageInputTokenDetails{
			TextTokens:  int64(meta.RequestUsage.InputTokens - meta.RequestUsage.AudioInputTokens),
			AudioTokens: int64(meta.RequestUsage.AudioInputTokens),
		},
		TotalTokens: int64(meta.RequestUsage.InputTokens) + outputTokens,
	}

	if strings.Contains(resp.Header.Get("Content-Type"), "json") {
		node, err := sonic.Get(responseBody)
		if err != nil {
			return usage.ToModelUsage(), relaymodel.WrapperOpenAIError(
				err,
				"get_node_from_body_err",
				http.StatusInternalServerError,
			)
		}

		if node.Get("usage").Exists() {
			usageStr, err := node.Get("usage").Raw()
			if err != nil {
				return usage.ToModelUsage(), relaymodel.WrapperOpenAIError(
					err,
					"unmarshal_response_err",
					http.StatusInternalServerError,
				)
			}

			err = sonic.UnmarshalString(usageStr, usage)
			if err != nil {
				return usage.ToModelUsage(), relaymodel.WrapperOpenAIError(
					err,
					"unmarshal_response_err",
					http.StatusInternalServerError,
				)
			}
		} else {
			_, err = node.SetAny("usage", usage)
			if err != nil {
				return usage.ToModelUsage(), relaymodel.WrapperOpenAIError(
					err,
					"marshal_response_err",
					http.StatusInternalServerError,
				)
			}

			responseBody, err = node.MarshalJSON()
			if err != nil {
				return usage.ToModelUsage(), relaymodel.WrapperOpenAIError(
					err,
					"marshal_response_err",
					http.StatusInternalServerError,
				)
			}
		}

		c.Writer.Header().Set("Content-Type", "application/json")
	}

	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(responseBody)))

	_, err = c.Writer.Write(responseBody)
	if err != nil {
		log.Warnf("write response body failed: %v", err)
	}

	return usage.ToModelUsage(), nil
}

func sttStreamHandler(
	meta *meta.Meta,
	c *gin.Context,
	resp *http.Response,
) (model.Usage, adaptor.Error) {
	defer resp.Body.Close()

	log := common.GetLogger(c)

	scanner := bufio.NewScanner(resp.Body)

	buf := GetScannerBuffer()
	defer PutScannerBuffer(buf)

	scanner.Buffer(*buf, cap(*buf))

	var (
		totalUsage *relaymodel.SttUsage
		fullText   strings.Builder
	)

	for scanner.Scan() {
		data := scanner.Bytes()
		if len(data) < DataPrefixLength { // ignore blank line or wrong format
			continue
		}

		if !slices.Equal(data[:DataPrefixLength], DataPrefixBytes) {
			continue
		}

		data = bytes.TrimSpace(data[DataPrefixLength:])
		if slices.Equal(data, DoneBytes) {
			break
		}

		var sseResponse relaymodel.SttSSEResponse

		err := sonic.Unmarshal(data, &sseResponse)
		if err != nil {
			log.Error("error unmarshalling STT stream response: " + err.Error())
			continue
		}

		switch sseResponse.Type {
		case relaymodel.SttSSEResponseTypeTranscriptTextDelta:
			if sseResponse.Delta != "" {
				fullText.WriteString(sseResponse.Delta)
			}
		case relaymodel.SttSSEResponseTypeTranscriptTextDone:
			if sseResponse.Usage != nil {
				fullText.Reset()

				totalUsage = sseResponse.Usage
			} else {
				text := fullText.String()
				fullText.Reset()

				if sseResponse.Text != "" {
					text = sseResponse.Text
				}

				outputTokens := CountTokenText(text, meta.ActualModel)
				totalUsage = &relaymodel.SttUsage{
					Type:         relaymodel.SttUsageTypeTokens,
					Seconds:      int64(meta.RequestUsage.AudioInputTokens),
					InputTokens:  int64(meta.RequestUsage.InputTokens),
					OutputTokens: outputTokens,
					InputTokenDetails: &relaymodel.SttUsageInputTokenDetails{
						TextTokens: int64(
							meta.RequestUsage.InputTokens - meta.RequestUsage.AudioInputTokens,
						),
						AudioTokens: int64(meta.RequestUsage.AudioInputTokens),
					},
					TotalTokens: int64(meta.RequestUsage.InputTokens) + outputTokens,
				}

				node, err := sonic.Get(data)
				if err != nil {
					return totalUsage.ToModelUsage(), relaymodel.WrapperOpenAIError(
						err,
						"get_node_from_body_err",
						http.StatusInternalServerError,
					)
				}

				_, err = node.SetAny("usage", totalUsage)
				if err != nil {
					return totalUsage.ToModelUsage(), relaymodel.WrapperOpenAIError(
						err,
						"marshal_response_err",
						http.StatusInternalServerError,
					)
				}

				data, err = node.MarshalJSON()
				if err != nil {
					return totalUsage.ToModelUsage(), relaymodel.WrapperOpenAIError(
						err,
						"marshal_response_err",
						http.StatusInternalServerError,
					)
				}
			}
		}

		BytesData(c, data)
	}

	Done(c)

	if err := scanner.Err(); err != nil {
		log.Error("error reading STT stream: " + err.Error())
	}

	// If no usage was provided, calculate based on text
	if totalUsage == nil {
		text := fullText.String()
		outputTokens := CountTokenText(text, meta.ActualModel)
		totalUsage = &relaymodel.SttUsage{
			Type:         relaymodel.SttUsageTypeTokens,
			Seconds:      int64(meta.RequestUsage.AudioInputTokens),
			InputTokens:  int64(meta.RequestUsage.InputTokens),
			OutputTokens: outputTokens,
			InputTokenDetails: &relaymodel.SttUsageInputTokenDetails{
				TextTokens: int64(
					meta.RequestUsage.InputTokens - meta.RequestUsage.AudioInputTokens,
				),
				AudioTokens: int64(meta.RequestUsage.AudioInputTokens),
			},
			TotalTokens: int64(meta.RequestUsage.InputTokens) + outputTokens,
		}
	}

	return totalUsage.ToModelUsage(), nil
}

func getTextFromVTT(body []byte) (string, error) {
	return getTextFromSRT(body)
}

func getTextFromVerboseJSON(body []byte) (string, error) {
	var whisperResponse relaymodel.SttVerboseJSONResponse
	if err := sonic.Unmarshal(body, &whisperResponse); err != nil {
		return "", fmt.Errorf("unmarshal_response_body_failed err :%w", err)
	}

	return whisperResponse.Text, nil
}

func getTextFromSRT(body []byte) (string, error) {
	scanner := bufio.NewScanner(bytes.NewReader(body))

	var (
		builder  strings.Builder
		textLine bool
	)

	for scanner.Scan() {
		line := scanner.Text()
		if textLine {
			builder.WriteString(line)

			textLine = false
			continue
		} else if strings.Contains(line, "-->") {
			textLine = true
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return builder.String(), nil
}

func getTextFromText(body []byte) string {
	return strings.TrimSuffix(conv.BytesToString(body), "\n")
}

func getTextFromJSON(body []byte) (string, error) {
	var whisperResponse relaymodel.SttJSONResponse
	if err := sonic.Unmarshal(body, &whisperResponse); err != nil {
		return "", fmt.Errorf("unmarshal_response_body_failed err :%w", err)
	}

	return whisperResponse.Text, nil
}
