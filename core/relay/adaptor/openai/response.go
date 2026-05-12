package openai

import (
	"bytes"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/meta"
	"github.com/labring/aiproxy/core/relay/mode"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
	"github.com/labring/aiproxy/core/relay/render"
	"github.com/labring/aiproxy/core/relay/utils"
	log "github.com/sirupsen/logrus"
)

const responseStreamInitialBufferTimeout = 2 * time.Second

// ConvertResponseRequest converts a response creation request
func ConvertResponseRequest(
	meta *meta.Meta,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	node, err := common.UnmarshalRequest2NodeReusable(req)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	// Set the model
	_, err = node.Set("model", ast.NewString(meta.ActualModel))
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	jsonData, err := node.MarshalJSON()
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	return adaptor.ConvertResult{
		Header: http.Header{
			"Content-Type":   {"application/json"},
			"Content-Length": {strconv.Itoa(len(jsonData))},
		},
		Body: bytes.NewReader(jsonData),
	}, nil
}

// ResponseHandler handles non-streaming response
func ResponseHandler(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if !adaptor.IsSuccessfulResponseStatus(mode.Responses, resp.StatusCode) {
		return adaptor.DoResponseResult{}, ErrorHanlder(resp)
	}

	defer resp.Body.Close()

	responseBody, err := common.GetResponseBody(resp)
	if err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIError(
			err,
			"read_response_body_failed",
			http.StatusInternalServerError,
		)
	}

	// Parse the response
	var response relaymodel.Response

	err = sonic.Unmarshal(responseBody, &response)
	if err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIError(
			err,
			"unmarshal_response_body_failed",
			http.StatusInternalServerError,
		)
	}

	if retryErr := responseRetryableError(response.Error); retryErr != nil {
		return adaptor.DoResponseResult{
			Usage:      response.ToModelUsage(),
			UpstreamID: response.ID,
		}, retryErr
	}

	// Store the response ID if needed for later retrieval
	if response.Store && response.ID != "" {
		err = store.SaveStore(adaptor.StoreCache{
			ID:        model.ResponseStoreID(response.ID),
			GroupID:   meta.Group.ID,
			TokenID:   meta.Token.ID,
			ChannelID: meta.Channel.ID,
			Model:     meta.OriginModel,
			ExpiresAt: time.Now().Add(time.Hour * 24 * 7), // Store for 7 days
		})
		if err != nil {
			log := common.GetLogger(c)
			log.Errorf("save response store failed: %v", err)
		}
	}

	// Write response
	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(responseBody)))
	_, _ = c.Writer.Write(responseBody)

	// Calculate usage
	usage := response.ToModelUsage()

	return adaptor.DoResponseResult{
		Usage:      usage,
		UpstreamID: response.ID,
		AsyncUsage: responseNeedsAsyncUsage(&response),
	}, nil
}

// ResponseStreamHandler handles streaming response
func ResponseStreamHandler(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if !adaptor.IsSuccessfulResponseStatus(mode.Responses, resp.StatusCode) {
		return adaptor.DoResponseResult{}, ErrorHanlder(resp)
	}

	defer resp.Body.Close()

	log := common.GetLogger(c)

	done := make(chan struct{})
	defer close(done)

	events := scanResponseStreamEvents(resp.Body, meta.OriginModel, meta.ActualModel, done)

	var (
		usage          model.Usage
		responseID     string
		lastResponse   *relaymodel.Response
		delayedEvents  [][]byte
		streamingBegan bool
		storeSaved     bool
		bufferTimer    *time.Timer
	)
	defer func() {
		stopResponseStreamBufferTimer(bufferTimer)
	}()

readLoop:
	for {
		var item responseStreamEventItem

		if bufferTimer == nil {
			next, ok := <-events
			if !ok {
				break
			}

			item = next
		} else {
			select {
			case next, ok := <-events:
				if !ok {
					break readLoop
				}

				item = next
			case <-bufferTimer.C:
				bufferTimer = nil
				flushDelayedResponseStreamEvents(c, &delayedEvents, &streamingBegan)
				continue
			}
		}

		if item.scanErr != nil {
			log.Error("error reading response stream: " + item.scanErr.Error())
			continue
		}

		if item.parseErr != nil {
			log.Error("error unmarshalling response stream: " + item.parseErr.Error())
			continue
		}

		event := item.event
		data := item.data

		if event.Response != nil {
			if responseID == "" {
				responseID = event.Response.ID
			}

			lastResponse = event.Response
			usage = event.Response.ToModelUsage()

			storeSaved = saveResponseStreamStoreIfNeeded(
				store,
				meta,
				event.Response,
				responseID,
				storeSaved,
				log,
			)
		}

		if retryErr := responseStreamRetryableError(&event); retryErr != nil && !streamingBegan {
			stopResponseStreamBufferTimer(bufferTimer)
			bufferTimer = nil

			return adaptor.DoResponseResult{
				Usage:      usage,
				UpstreamID: responseID,
				AsyncUsage: responseNeedsAsyncUsage(lastResponse),
			}, retryErr
		}

		if responseStreamEventCanDelay(event.Type) && !streamingBegan {
			delayedEvents = append(delayedEvents, append([]byte(nil), data...))
			if bufferTimer == nil {
				bufferTimer = time.NewTimer(responseStreamInitialBufferTimeout)
			}

			continue
		}

		stopResponseStreamBufferTimer(bufferTimer)
		bufferTimer = nil
		flushDelayedResponseStreamEvents(c, &delayedEvents, &streamingBegan)

		// Forward the event
		render.ResponsesData(c, data)
		streamingBegan = true
	}

	flushDelayedResponseStreamEvents(c, &delayedEvents, &streamingBegan)

	return adaptor.DoResponseResult{
		Usage:      usage,
		UpstreamID: responseID,
		AsyncUsage: responseNeedsAsyncUsage(lastResponse),
	}, nil
}

func responseNeedsAsyncUsage(response *relaymodel.Response) bool {
	if response == nil || response.ID == "" || response.Usage != nil {
		return false
	}

	if usage := response.ToModelUsage(); usage.TotalTokens > 0 || usage.WebSearchCount > 0 {
		return false
	}

	switch response.Status {
	case relaymodel.ResponseStatusInProgress, relaymodel.ResponseStatusQueued:
		return true
	default:
		return false
	}
}

func saveResponseStreamStoreIfNeeded(
	store adaptor.Store,
	meta *meta.Meta,
	response *relaymodel.Response,
	responseID string,
	storeSaved bool,
	log *log.Entry,
) bool {
	if storeSaved || response == nil || !response.Store || responseID == "" {
		return storeSaved
	}

	saveErr := store.SaveStore(adaptor.StoreCache{
		ID:        model.ResponseStoreID(responseID),
		GroupID:   meta.Group.ID,
		TokenID:   meta.Token.ID,
		ChannelID: meta.Channel.ID,
		Model:     meta.OriginModel,
		ExpiresAt: time.Now().Add(time.Hour * 24 * 7),
	})
	if saveErr != nil {
		log.Errorf("save response store failed: %v", saveErr)
	}

	return true
}

type responseStreamEventItem struct {
	data     []byte
	event    relaymodel.ResponseStreamEvent
	parseErr error
	scanErr  error
}

func scanResponseStreamEvents(
	body io.Reader,
	originModel string,
	actualModel string,
	done <-chan struct{},
) <-chan responseStreamEventItem {
	events := make(chan responseStreamEventItem, 16)

	go func() {
		defer close(events)

		scanner, cleanup := utils.NewStreamScanner(body, originModel, actualModel)
		defer cleanup()

		for scanner.Scan() {
			data := scanner.Bytes()
			if !render.IsValidSSEData(data) {
				continue
			}

			data = append([]byte(nil), render.ExtractSSEData(data)...)

			var event relaymodel.ResponseStreamEvent
			parseErr := sonic.Unmarshal(data, &event)

			item := responseStreamEventItem{
				data:     data,
				event:    event,
				parseErr: parseErr,
			}

			select {
			case events <- item:
			case <-done:
				return
			}
		}

		if err := scanner.Err(); err != nil {
			select {
			case events <- responseStreamEventItem{scanErr: err}:
			case <-done:
			}
		}
	}()

	return events
}

func stopResponseStreamBufferTimer(timer *time.Timer) {
	if timer == nil {
		return
	}

	if !timer.Stop() {
		select {
		case <-timer.C:
		default:
		}
	}
}

func flushDelayedResponseStreamEvents(
	c *gin.Context,
	delayedEvents *[][]byte,
	streamingBegan *bool,
) {
	if len(*delayedEvents) == 0 {
		return
	}

	for _, delayedEvent := range *delayedEvents {
		render.ResponsesData(c, delayedEvent)
	}

	*delayedEvents = nil
	*streamingBegan = true
}

func responseStreamEventCanDelay(eventType string) bool {
	switch eventType {
	case relaymodel.EventResponseCreated,
		relaymodel.EventResponseInProgress,
		relaymodel.EventResponseQueued,
		"keepalive":
		return true
	default:
		return false
	}
}

func responseStreamRetryableError(event *relaymodel.ResponseStreamEvent) adaptor.Error {
	if event == nil {
		return nil
	}

	if event.Response != nil && event.Response.Status == relaymodel.ResponseStatusFailed {
		return responseRetryableError(event.Response.Error)
	}

	if event.Type == relaymodel.EventError {
		return responseRetryableError(&relaymodel.ResponseError{
			Code:    event.Code,
			Message: event.Message,
		})
	}

	return nil
}

func responseRetryableError(respErr *relaymodel.ResponseError) adaptor.Error {
	if respErr == nil {
		return nil
	}

	statusCode := http.StatusInternalServerError
	switch respErr.Code {
	case "server_is_overloaded", "slow_down":
		statusCode = http.StatusServiceUnavailable
	case "server_error", "internal_server_error":
	default:
		return nil
	}

	message := respErr.Message
	if message == "" {
		message = "upstream response failed with retryable server error"
	}

	return relaymodel.NewOpenAIError(statusCode, relaymodel.OpenAIError{
		Message: message,
		Type:    "server_error",
		Code:    respErr.Code,
	})
}

// GetResponseHandler handles GET /v1/responses/{response_id}
func GetResponseHandler(
	_ *meta.Meta,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return adaptor.DoResponseResult{}, ErrorHanlder(resp)
	}

	defer resp.Body.Close()

	c.Writer.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	c.Writer.Header().Set("Content-Length", resp.Header.Get("Content-Length"))
	_, _ = io.Copy(c.Writer, resp.Body)

	return adaptor.DoResponseResult{}, nil
}

// DeleteResponseHandler handles DELETE /v1/responses/{response_id}
func DeleteResponseHandler(
	_ *meta.Meta,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if !adaptor.IsSuccessfulResponseStatus(mode.ResponsesDelete, resp.StatusCode) {
		return adaptor.DoResponseResult{}, ErrorHanlder(resp)
	}

	defer resp.Body.Close()

	c.Writer.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	c.Writer.Header().Set("Content-Length", resp.Header.Get("Content-Length"))
	c.Status(http.StatusNoContent)
	_, _ = io.Copy(c.Writer, resp.Body)

	return adaptor.DoResponseResult{}, nil
}

// CancelResponseHandler handles POST /v1/responses/{response_id}/cancel
func CancelResponseHandler(
	_ *meta.Meta,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return adaptor.DoResponseResult{}, ErrorHanlder(resp)
	}

	defer resp.Body.Close()

	c.Writer.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	c.Writer.Header().Set("Content-Length", resp.Header.Get("Content-Length"))
	_, _ = io.Copy(c.Writer, resp.Body)

	return adaptor.DoResponseResult{}, nil
}

// GetInputItemsHandler handles GET /v1/responses/{response_id}/input_items
func GetInputItemsHandler(
	_ *meta.Meta,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return adaptor.DoResponseResult{}, ErrorHanlder(resp)
	}

	defer resp.Body.Close()

	c.Writer.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	c.Writer.Header().Set("Content-Length", resp.Header.Get("Content-Length"))
	_, _ = io.Copy(c.Writer, resp.Body)

	return adaptor.DoResponseResult{}, nil
}
