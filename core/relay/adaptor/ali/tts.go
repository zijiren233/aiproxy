package ali

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/adaptor/openai"
	"github.com/labring/aiproxy/core/relay/meta"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/labring/aiproxy/core/relay/utils"
)

type TTSMessage struct {
	Header  TTSHeader  `json:"header"`
	Payload TTSPayload `json:"payload"`
}

type TTSHeader struct {
	Attributes   map[string]any `json:"attributes"`
	Action       string         `json:"action,omitempty"`
	TaskID       string         `json:"task_id"`
	Streaming    string         `json:"streaming,omitempty"`
	Event        string         `json:"event,omitempty"`
	ErrorCode    string         `json:"error_code,omitempty"`
	ErrorMessage string         `json:"error_message,omitempty"`
}

type TTSPayload struct {
	Model      string        `json:"model,omitempty"`
	TaskGroup  string        `json:"task_group,omitempty"`
	Task       string        `json:"task,omitempty"`
	Function   string        `json:"function,omitempty"`
	Input      TTSInput      `json:"input,omitempty"`
	Output     TTSOutput     `json:"output,omitempty"`
	Parameters TTSParameters `json:"parameters,omitempty"`
	Usage      TTSUsage      `json:"usage,omitempty"`
}

type TTSInput struct {
	Text string `json:"text"`
}

type TTSParameters struct {
	TextType                string  `json:"text_type"`
	Format                  string  `json:"format"`
	SampleRate              int     `json:"sample_rate,omitempty"`
	Volume                  int     `json:"volume"`
	Rate                    float64 `json:"rate"`
	Pitch                   float64 `json:"pitch"`
	WordTimestampEnabled    bool    `json:"word_timestamp_enabled"`
	PhonemeTimestampEnabled bool    `json:"phoneme_timestamp_enabled"`
}

type TTSOutput struct {
	Sentence TTSSentence `json:"sentence"`
}

type TTSSentence struct {
	Words     []TTSWord `json:"words"`
	BeginTime int       `json:"begin_time"`
	EndTime   int       `json:"end_time"`
}

type TTSWord struct {
	Text      string       `json:"text"`
	Phonemes  []TTSPhoneme `json:"phonemes"`
	BeginTime int          `json:"begin_time"`
	EndTime   int          `json:"end_time"`
}

type TTSPhoneme struct {
	Text      string `json:"text"`
	BeginTime int    `json:"begin_time"`
	EndTime   int    `json:"end_time"`
	Tone      int    `json:"tone"`
}

type TTSUsage struct {
	Characters int64 `json:"characters"`
}

var ttsSupportedFormat = map[string]struct{}{
	"pcm": {},
	"wav": {},
	"mp3": {},
}

func ConvertTTSRequest(meta *meta.Meta, req *http.Request) (adaptor.ConvertResult, error) {
	request, err := utils.UnmarshalTTSRequest(req)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	reqMap, err := utils.UnmarshalMap(req)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	var sampleRate int

	sampleRateI, ok := reqMap["sample_rate"].(float64)
	if ok {
		sampleRate = int(sampleRateI)
	}

	request.Model = meta.ActualModel

	meta.Set("stream_format", request.StreamFormat)

	if strings.HasPrefix(request.Model, "sambert-v") {
		voice := request.Voice
		if voice == "" {
			voice = "zhinan"
		}

		request.Model = fmt.Sprintf(
			"sambert-%s-v%s",
			voice,
			strings.TrimPrefix(request.Model, "sambert-v"),
		)
	}

	ttsRequest := TTSMessage{
		Header: TTSHeader{
			Action:    "run-task",
			Streaming: "out",
			TaskID:    uuid.NewString(),
		},
		Payload: TTSPayload{
			Model:     request.Model,
			Task:      "tts",
			TaskGroup: "audio",
			Function:  "SpeechSynthesizer",
			Input: TTSInput{
				Text: request.Input,
			},
			Parameters: TTSParameters{
				TextType:                "PlainText",
				Format:                  "wav",
				Volume:                  50,
				SampleRate:              sampleRate,
				Rate:                    request.Speed,
				Pitch:                   1.0,
				WordTimestampEnabled:    true,
				PhonemeTimestampEnabled: true,
			},
		},
	}

	if _, ok := ttsSupportedFormat[request.ResponseFormat]; ok {
		ttsRequest.Payload.Parameters.Format = request.ResponseFormat
	}

	if ttsRequest.Payload.Parameters.Rate < 0.5 {
		ttsRequest.Payload.Parameters.Rate = 0.5
	} else if ttsRequest.Payload.Parameters.Rate > 2 {
		ttsRequest.Payload.Parameters.Rate = 2
	}

	data, err := sonic.Marshal(ttsRequest)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	return adaptor.ConvertResult{
		Header: http.Header{
			"X-DashScope-DataInspection": {"enable"},
		},
		Body: bytes.NewReader(data),
	}, nil
}

func TTSDoRequest(meta *meta.Meta, req *http.Request) (*http.Response, error) {
	wsURL := req.URL
	wsURL.Scheme = "wss"

	conn, _, err := websocket.DefaultDialer.Dial(wsURL.String(), req.Header)
	if err != nil {
		return nil, err
	}

	meta.Set("ws_conn", conn)

	writer, err := conn.NextWriter(websocket.TextMessage)
	if err != nil {
		return nil, err
	}
	defer writer.Close()

	_, err = io.Copy(writer, req.Body)
	if err != nil {
		return nil, err
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(nil),
	}, nil
}

func TTSDoResponse(
	meta *meta.Meta,
	c *gin.Context,
	_ *http.Response,
) (usage model.Usage, err adaptor.Error) {
	log := common.GetLogger(c)

	conn, ok := meta.MustGet("ws_conn").(*websocket.Conn)
	if !ok {
		panic(fmt.Sprintf("ws conn type error: %T, %v", conn, conn))
	}
	defer conn.Close()

	sseFormat := meta.GetString("stream_format") == "sse"

	usage = model.Usage{}

	for {
		messageType, data, err := conn.ReadMessage()
		if err != nil {
			return usage, relaymodel.WrapperOpenAIErrorWithMessage(
				"ali_wss_read_msg_failed",
				nil,
				http.StatusInternalServerError,
			)
		}

		var msg TTSMessage
		switch messageType {
		case websocket.TextMessage:
			err = sonic.Unmarshal(data, &msg)
			if err != nil {
				return usage, relaymodel.WrapperOpenAIErrorWithMessage(
					"ali_wss_read_msg_failed",
					nil,
					http.StatusInternalServerError,
				)
			}

			switch msg.Header.Event {
			case "task-started":
				continue
			case "result-generated":
				continue
			case "task-finished":
				usage.InputTokens = model.ZeroNullInt64(msg.Payload.Usage.Characters)
				usage.TotalTokens = model.ZeroNullInt64(msg.Payload.Usage.Characters)
				return usage, nil
			case "task-failed":
				if sseFormat {
					openai.AudioDone(c, relaymodel.TextToSpeechUsage{
						InputTokens:  int64(usage.InputTokens),
						OutputTokens: int64(usage.OutputTokens),
						TotalTokens:  int64(usage.TotalTokens),
					})

					return usage, nil
				}

				return usage, relaymodel.WrapperOpenAIErrorWithMessage(
					msg.Header.ErrorMessage,
					msg.Header.ErrorCode,
					http.StatusInternalServerError,
				)
			}
		case websocket.BinaryMessage:
			if sseFormat {
				openai.AudioData(c, base64.StdEncoding.EncodeToString(data))
				continue
			}

			_, writeErr := c.Writer.Write(data)
			if writeErr != nil {
				log.Error("write tts response chunk failed: " + writeErr.Error())
			}
		}
	}
}
