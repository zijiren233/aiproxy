package openai

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/ast"
	"github.com/gin-gonic/gin"
	"github.com/labring/aiproxy/core/common"
	"github.com/labring/aiproxy/core/model"
	"github.com/labring/aiproxy/core/relay/adaptor"
	"github.com/labring/aiproxy/core/relay/meta"
	relaymodel "github.com/labring/aiproxy/core/relay/model"
)

func ConvertVideoRequest(
	meta *meta.Meta,
	req *http.Request,
) (adaptor.ConvertResult, error) {
	node, err := common.UnmarshalRequest2NodeReusable(req)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	_, err = node.Set("model", ast.NewString(meta.ActualModel))
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	jsonData, err := sonic.Marshal(&node)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	fullURL, err := url.JoinPath(meta.Channel.BaseURL, "/video/generations/jobs")
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	return adaptor.ConvertResult{
		Method: http.MethodPost,
		URL:    fullURL,
		Header: http.Header{
			"Content-Type":   {"application/json"},
			"Content-Length": {strconv.Itoa(len(jsonData))},
		},
		Body: bytes.NewReader(jsonData),
	}, nil
}

func ConvertVideoGetJobsRequest(
	meta *meta.Meta,
	_ *http.Request,
) (adaptor.ConvertResult, error) {
	fullURL, err := url.JoinPath(meta.Channel.BaseURL, "/video/generations/jobs", meta.JobID)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	return adaptor.ConvertResult{
		Method: http.MethodGet,
		URL:    fullURL,
	}, nil
}

func ConvertVideoGetJobsContentRequest(
	meta *meta.Meta,
	_ *http.Request,
) (adaptor.ConvertResult, error) {
	fullURL, err := url.JoinPath(
		meta.Channel.BaseURL,
		"/video/generations",
		meta.GenerationID,
		"/content/video",
	)
	if err != nil {
		return adaptor.ConvertResult{}, err
	}

	return adaptor.ConvertResult{
		Method: http.MethodGet,
		URL:    fullURL,
	}, nil
}

func VideoHandler(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (model.Usage, adaptor.Error) {
	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusCreated {
		return model.Usage{}, VideoErrorHanlder(resp)
	}

	defer resp.Body.Close()

	responseBody, err := common.GetResponseBody(resp)
	if err != nil {
		return model.Usage{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	idNode, err := sonic.GetWithOptions(responseBody, ast.SearchOptions{}, "id")
	if err != nil {
		return model.Usage{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	id, err := idNode.String()
	if err != nil {
		return model.Usage{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	err = store.SaveStore(adaptor.StoreCache{
		ID:        id,
		GroupID:   meta.Group.ID,
		TokenID:   meta.Token.ID,
		ChannelID: meta.Channel.ID,
		Model:     meta.ActualModel,
		ExpiresAt: time.Now().Add(time.Hour * 24),
	})
	if err != nil {
		log := common.GetLogger(c)
		log.Errorf("save store failed: %v", err)
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(responseBody)))
	_, _ = c.Writer.Write(responseBody)

	return model.Usage{}, nil
}

// VideoHandlerWithUsageResult handles video generation job creation and returns async usage info
func VideoHandlerWithUsageResult(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (adaptor.UsageResult, adaptor.Error) {
	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusCreated {
		return nil, VideoErrorHanlder(resp)
	}

	defer resp.Body.Close()

	responseBody, err := common.GetResponseBody(resp)
	if err != nil {
		return nil, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	idNode, err := sonic.GetWithOptions(responseBody, ast.SearchOptions{}, "id")
	if err != nil {
		return nil, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	id, err := idNode.String()
	if err != nil {
		return nil, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	err = store.SaveStore(adaptor.StoreCache{
		ID:        id,
		GroupID:   meta.Group.ID,
		TokenID:   meta.Token.ID,
		ChannelID: meta.Channel.ID,
		Model:     meta.ActualModel,
		ExpiresAt: time.Now().Add(time.Hour * 24),
	})
	if err != nil {
		log := common.GetLogger(c)
		log.Errorf("save store failed: %v", err)
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(responseBody)))
	_, _ = c.Writer.Write(responseBody)

	// Return async usage result with job ID in Data field
	return adaptor.NewAsyncUsage(&model.AsyncUsageInfo{
		Mode:      int(meta.Mode),
		Model:     meta.ActualModel,
		ChannelID: meta.Channel.ID,
		GroupID:   meta.Group.ID,
		TokenID:   meta.Token.ID,
		Data:      `{"job_id":"` + id + `"}`,
	}), nil
}

func VideoGetJobsHandler(
	meta *meta.Meta,
	store adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (model.Usage, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return model.Usage{}, VideoErrorHanlder(resp)
	}

	defer resp.Body.Close()

	responseBody, err := common.GetResponseBody(resp)
	if err != nil {
		return model.Usage{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	node, err := sonic.Get(responseBody)
	if err != nil {
		return model.Usage{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	expiresAt, err := node.Get("expires_at").Int64()
	if err != nil {
		return model.Usage{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	generationsNode := node.Get("generations")

	var patchErr error

	err = generationsNode.ForEach(func(_ ast.Sequence, node *ast.Node) bool {
		idNode := node.Get("id")

		id, err := idNode.String()
		if err != nil {
			patchErr = err
			return false
		}

		err = store.SaveStore(adaptor.StoreCache{
			ID:        id,
			GroupID:   meta.Group.ID,
			TokenID:   meta.Token.ID,
			ChannelID: meta.Channel.ID,
			Model:     meta.ActualModel,
			ExpiresAt: time.Unix(expiresAt, 0),
		})
		if err != nil {
			log := common.GetLogger(c)
			log.Errorf("save store failed: %v", err)
		}

		return true
	})
	if err == nil {
		err = patchErr
	}

	if err != nil {
		return model.Usage{}, relaymodel.WrapperOpenAIVideoError(
			err,
			http.StatusInternalServerError,
		)
	}

	c.Writer.Header().Set("Content-Type", "application/json")
	c.Writer.Header().Set("Content-Length", strconv.Itoa(len(responseBody)))
	_, _ = c.Writer.Write(responseBody)

	return model.Usage{}, nil
}

func VideoGetJobsContentHandler(
	_ *meta.Meta,
	_ adaptor.Store,
	c *gin.Context,
	resp *http.Response,
) (model.Usage, adaptor.Error) {
	if resp.StatusCode != http.StatusOK {
		return model.Usage{}, VideoErrorHanlder(resp)
	}

	defer resp.Body.Close()

	c.Writer.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
	c.Writer.Header().Set("Content-Length", resp.Header.Get("Content-Length"))
	_, _ = io.Copy(c.Writer, resp.Body)

	return model.Usage{}, nil
}
