package controller

import (
	"context"
	"errors"
	"time"

	"github.com/bytedance/sonic"
	"github.com/labring/aiproxy/core/model"
	mcpservers "github.com/labring/aiproxy/mcp-servers"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

func getPublicMCPTools(
	ctx context.Context,
	publicMcp model.PublicMCP,
	params map[string]string,
) ([]mcp.Tool, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	switch publicMcp.Type {
	case model.PublicMCPTypeEmbed:
		return getEmbedMCPTools(ctx, publicMcp, params)
	case model.PublicMCPTypeOpenAPI:
		return getOpenAPIMCPTools(ctx, publicMcp)
	case model.PublicMCPTypeProxySSE:
		return getProxySSEMCPTools(ctx, publicMcp, params)
	case model.PublicMCPTypeProxyStreamable:
		return getProxyStreamableMCPTools(ctx, publicMcp, params)
	default:
		return nil, nil
	}
}

func getEmbedMCPTools(
	ctx context.Context,
	publicMcp model.PublicMCP,
	params map[string]string,
) ([]mcp.Tool, error) {
	if publicMcp.EmbedConfig == nil {
		return nil, nil
	}

	server, err := mcpservers.GetMCPServer(publicMcp.ID, publicMcp.EmbedConfig.Init, params)
	if err != nil {
		return nil, err
	}

	return getMCPServerTools(ctx, server)
}

func getOpenAPIMCPTools(ctx context.Context, publicMcp model.PublicMCP) ([]mcp.Tool, error) {
	if publicMcp.OpenAPIConfig == nil {
		return nil, nil
	}

	server, err := newOpenAPIMCPServer(publicMcp.OpenAPIConfig)
	if err != nil {
		return nil, err
	}

	return getMCPServerTools(ctx, server)
}

func getProxySSEMCPTools(
	ctx context.Context,
	publicMcp model.PublicMCP,
	params map[string]string,
) ([]mcp.Tool, error) {
	if publicMcp.ProxyConfig == nil {
		return nil, nil
	}

	url, headers, err := prepareProxyConfig(publicMcp.ToPublicMCPCache(), staticParams(params))
	if err != nil {
		return nil, err
	}
	client, err := transport.NewSSE(url, transport.WithHeaders(headers))
	if err != nil {
		return nil, err
	}
	defer client.Close()

	if err := client.Start(ctx); err != nil {
		return nil, err
	}

	return getMCPServerTools(ctx, mcpservers.WrapMCPClient2Server(client))
}

func getProxyStreamableMCPTools(
	ctx context.Context,
	publicMcp model.PublicMCP,
	params map[string]string,
) ([]mcp.Tool, error) {
	if publicMcp.ProxyConfig == nil {
		return nil, nil
	}

	url, headers, err := prepareProxyConfig(publicMcp.ToPublicMCPCache(), staticParams(params))
	if err != nil {
		return nil, err
	}

	client, err := transport.NewStreamableHTTP(
		url,
		transport.WithHTTPHeaders(headers),
	)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	if err := client.Start(ctx); err != nil {
		return nil, err
	}

	return getMCPServerTools(ctx, mcpservers.WrapMCPClient2Server(client))
}

func getMCPServerTools(ctx context.Context, server mcpservers.Server) ([]mcp.Tool, error) {
	requestBytes, err := sonic.Marshal(mcp.JSONRPCRequest{
		JSONRPC: mcp.JSONRPC_VERSION,
		ID:      mcp.NewRequestId(1),
		Request: mcp.Request{
			Method: string(mcp.MethodToolsList),
			Params: mcp.RequestParams{},
		},
	})
	if err != nil {
		return nil, err
	}

	response := server.HandleMessage(ctx, requestBytes)
	if response == nil {
		return nil, errors.New("no response from server")
	}

	responseBytes, err := sonic.Marshal(response)
	if err != nil {
		return nil, err
	}

	var jsonRPCResponse struct {
		Result *mcp.ListToolsResult `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := sonic.Unmarshal(responseBytes, &jsonRPCResponse); err != nil {
		return nil, err
	}

	if jsonRPCResponse.Error != nil {
		return nil, errors.New(jsonRPCResponse.Error.Message)
	}

	if jsonRPCResponse.Result == nil {
		return nil, nil
	}

	return jsonRPCResponse.Result.Tools, nil
}
