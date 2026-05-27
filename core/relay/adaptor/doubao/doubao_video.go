package doubao

import (
	"bytes"
	"net/http"
	"strconv"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/meta"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
)

func ConvertDoubaoNativeVideoRequest(
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
			"Content-Type":   {"application/json"},
			"Content-Length": {strconv.Itoa(len(data))},
		},
		Body: bytes.NewReader(data),
	}, nil
}

func DoubaoNativeVideoSubmitHandler(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	body, response, relayErr := readDoubaoNativeVideoTaskResponse(resp, true)
	if relayErr != nil {
		return adaptor.DoResponseResult{}, relayErr
	}

	expiresAt := doubaoVideoExpiresAt(response)
	if err := saveDoubaoVideoStore(meta, store, response.ID, expiresAt); err != nil {
		common.GetLogger(c).Errorf("save doubao native video store failed: %v", err)
	}

	writeDoubaoNativeJSONResponse(c, resp, body)

	return adaptor.DoResponseResult{
		UpstreamID: response.ID,
		AsyncUsage: true,
		UsageContext: doubaoVideoUsageContext(
			&response,
		).WithFallback(doubaoVideoRequestUsageContext(meta)),
	}, nil
}

func DoubaoNativeVideoTaskHandler(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	body, response, relayErr := readDoubaoNativeVideoTaskResponse(resp, false)
	if relayErr != nil {
		return adaptor.DoResponseResult{}, relayErr
	}

	if response.ID == "" {
		response.ID = meta.VideoID
	}

	if response.ID != "" {
		expiresAt := doubaoVideoExpiresAt(response)
		if err := saveDoubaoVideoStore(meta, store, response.ID, expiresAt); err != nil {
			common.GetLogger(c).Errorf("save doubao native video store failed: %v", err)
		}
	}

	writeDoubaoNativeJSONResponse(c, resp, body)

	return adaptor.DoResponseResult{
		UpstreamID: response.ID,
		UsageContext: doubaoVideoUsageContext(
			&response,
		).WithFallback(doubaoVideoRequestUsageContext(meta)),
	}, nil
}

func DoubaoNativeVideoTaskDeleteHandler(
	_ *meta.Meta,
	c *gin.Context,
	resp *http.Response,
) (adaptor.DoResponseResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return adaptor.DoResponseResult{}, ErrorHandler(resp)
	}

	defer resp.Body.Close()

	body, err := common.GetResponseBody(resp)
	if err != nil {
		return adaptor.DoResponseResult{}, relaymodel.WrapperOpenAIErrorWithMessage(
			err.Error(),
			relaymodel.ErrorCodeBadResponse,
			resp.StatusCode,
			relaymodel.ErrorTypeUpstream,
		)
	}

	if resp.StatusCode == http.StatusNoContent {
		c.Writer.WriteHeader(http.StatusNoContent)
		return adaptor.DoResponseResult{}, nil
	}

	writeDoubaoNativeJSONResponse(c, resp, body)

	return adaptor.DoResponseResult{}, nil
}

func readDoubaoNativeVideoTaskResponse(
	resp *http.Response,
	requireID bool,
) ([]byte, relaymodel.DoubaoVideoTaskResponse, adaptor.Error) {
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, relaymodel.DoubaoVideoTaskResponse{}, ErrorHandler(resp)
	}

	defer resp.Body.Close()

	body, err := common.GetResponseBody(resp)
	if err != nil {
		return nil, relaymodel.DoubaoVideoTaskResponse{}, relaymodel.WrapperOpenAIErrorWithMessage(
			err.Error(),
			relaymodel.ErrorCodeBadResponse,
			resp.StatusCode,
			relaymodel.ErrorTypeUpstream,
		)
	}

	var response relaymodel.DoubaoVideoTaskResponse
	if err := sonic.Unmarshal(body, &response); err != nil {
		return nil, relaymodel.DoubaoVideoTaskResponse{}, relaymodel.WrapperOpenAIErrorWithMessage(
			err.Error(),
			relaymodel.ErrorCodeBadResponse,
			http.StatusInternalServerError,
			relaymodel.ErrorTypeUpstream,
		)
	}

	if requireID && response.ID == "" {
		return nil, relaymodel.DoubaoVideoTaskResponse{}, relaymodel.WrapperOpenAIErrorWithMessage(
			"missing id in doubao video response",
			relaymodel.ErrorCodeBadResponse,
			http.StatusInternalServerError,
			relaymodel.ErrorTypeUpstream,
		)
	}

	return body, response, nil
}

func writeDoubaoNativeJSONResponse(c *gin.Context, resp *http.Response, body []byte) {
	contentType := "application/json"
	if resp != nil && resp.Header.Get("Content-Type") != "" {
		contentType = resp.Header.Get("Content-Type")
	}

	c.Writer.Header().Set("Content-Type", contentType)
	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(body)))
	_, _ = c.Writer.Write(body)
}
