package ali

import (
	"bytes"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	coremodel "github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/meta"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
)

func ConvertAliNativeVideoRequest(
	meta *meta.Meta,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	body, err := common.UnmarshalRequest2NodeReusable(req)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	if _, err := body.Set("model", ast.NewString(meta.ActualModel)); err != nil {
		return adaptor.ConvertResult{}, err
	}

	data, err := body.MarshalJSON()
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	return adaptor.ConvertResult{
		Header: http.Header{
			"X-Dashscope-Async": {"enable"},
			"Content-Type":      {"application/json"},
			"Content-Length":    {strconv.Itoa(len(data))},
		},
		Body: bytes.NewReader(data),
	}, nil
}

func AliNativeVideoHandler(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	body, relayErr := readAliNativeVideoResponseBody(resp)
	if relayErr != nil {
		return adaptor.DoResponseResult{}, relayErr
	}

	var aliResponse relaymodel.AliVideoTaskResponse
	if err := sonic.Unmarshal(body, &aliResponse); err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIErrorWithMessage(
			err.Error(),
			relaymodel.ErrorCodeBadResponse,
			http.StatusInternalServerError,
			relaymodel.ErrorTypeUpstream,
		)
	}

	if aliResponse.Code != "" || aliResponse.Message != "" {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIErrorWithMessage(
			firstNonEmpty(aliResponse.Message, aliResponse.Code),
			relaymodel.ErrorCodeBadResponse,
			http.StatusInternalServerError,
			relaymodel.ErrorTypeUpstream,
		)
	}

	taskID := strings.TrimSpace(aliResponse.Output.TaskID)
	if taskID == "" {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIErrorWithMessage(
			"missing output.task_id in ali video response",
			relaymodel.ErrorCodeBadResponse,
			http.StatusInternalServerError,
			relaymodel.ErrorTypeUpstream,
		)
	}

	if err := saveAliNativeVideoStore(meta, store, taskID, aliVideoTaskExpiresAt(nil)); err != nil {
		common.GetLogger(c).Errorf("save ali native video store failed: %v", err)
	}

	writeAliNativeJSONResponse(c, resp, body)

	return adaptor.DoResponseResult{
		UpstreamID:   taskID,
		AsyncUsage:   true,
		UsageContext: aliVideoUsageContext(meta, aliResponse.Usage),
	}, nil
}

func AliNativeVideoTaskHandler(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	body, relayErr := readAliNativeVideoResponseBody(resp)
	if relayErr != nil {
		return adaptor.DoResponseResult{}, relayErr
	}

	var aliResponse relaymodel.AliVideoTaskResponse
	if err := sonic.Unmarshal(body, &aliResponse); err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIErrorWithMessage(
			err.Error(),
			relaymodel.ErrorCodeBadResponse,
			http.StatusInternalServerError,
			relaymodel.ErrorTypeUpstream,
		)
	}

	if aliResponse.Output.TaskID == "" {
		aliResponse.Output.TaskID = meta.VideoID
	}

	taskID := strings.TrimSpace(aliResponse.Output.TaskID)
	if taskID != "" {
		if err := saveAliNativeVideoStore(
			meta,
			store,
			taskID,
			aliVideoTaskExpiresAt(&aliResponse),
		); err != nil {
			common.GetLogger(c).Errorf("save ali native video store failed: %v", err)
		}
	}

	writeAliNativeJSONResponse(c, resp, body)

	return adaptor.DoResponseResult{
		UpstreamID:   firstNonEmpty(taskID, meta.VideoID),
		UsageContext: aliVideoUsageContext(meta, aliResponse.Usage),
	}, nil
}

func readAliNativeVideoResponseBody(resp *http.Response) ([]byte, adaptor.Error) {
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, ErrorHanlder(resp)
	}

	defer resp.Body.Close()

	body, err := common.GetResponseBody(resp)
	if err != nil {
		return nil, relaymodel.WrapperOpenAIErrorWithMessage(
			err.Error(),
			relaymodel.ErrorCodeBadResponse,
			resp.StatusCode,
			relaymodel.ErrorTypeUpstream,
		)
	}

	return body, nil
}

func saveAliNativeVideoStore(
	meta *meta.Meta,
	store adaptor.Store,
	taskID string,
	expiresAt time.Time,
) error {
	if store == nil || meta == nil || taskID == "" {
		return nil
	}

	return store.SaveStore(adaptor.StoreCache{
		ID:        coremodel.VideoGenerationStoreID(taskID),
		GroupID:   meta.Group.ID,
		TokenID:   meta.Token.ID,
		ChannelID: meta.Channel.ID,
		Model:     meta.OriginModel,
		ExpiresAt: expiresAt,
	})
}

func aliVideoTaskExpiresAt(response *relaymodel.AliVideoTaskResponse) time.Time {
	if response != nil && response.Output.EndTime != "" {
		if end, err := time.Parse(time.RFC3339, response.Output.EndTime); err == nil {
			return end.Add(aliVideoTaskTTL)
		}
	}

	return time.Now().Add(aliVideoTaskTTL)
}

func writeAliNativeJSONResponse(c *gin.Context, resp *http.Response, body []byte) {
	contentType := "application/json"
	if resp != nil && resp.Header.Get("Content-Type") != "" {
		contentType = resp.Header.Get("Content-Type")
	}

	c.Writer.Header().Set("Content-Type", contentType)
	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(body)))
	_, _ = c.Writer.Write(body)
}
