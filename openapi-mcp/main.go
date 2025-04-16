package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/mark3labs/mcp-go/server"
	"github.com/zijiren233/openapi-mcp/convert"
)

var (
	sse  string
	file string
	v2   bool
)

func init() {
	flag.StringVar(&sse, "sse", "", "it will use sse protocol, example: :3000")
	flag.StringVar(&file, "file", "", "openapi file path")
	flag.BoolVar(&v2, "v2", false, "openapi v2 version")
}

func getFronHTTP(u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func newServer() (*server.MCPServer, error) {
	parser := convert.NewParser()
	var err error

	if strings.HasPrefix(file, "http://") || strings.HasPrefix(file, "https://") {
		var content []byte
		content, err = getFronHTTP(file)
		if err != nil {
			return nil, err
		}

		if v2 {
			err = parser.ParseV2(content)
		} else {
			err = parser.Parse(content)
		}
	} else {
		// For local files, use ParseFile
		if v2 {
			err = parser.ParseFileV2(file)
		} else {
			err = parser.ParseFile(file)
		}
	}

	if err != nil {
		return nil, err
	}
	converter := convert.NewConverter(parser, convert.Options{})
	return converter.Convert()
}

func main() {
	flag.Parse()

	if file == "" {
		log.Fatal("Not provied openapi file")
	}

	s, err := newServer()
	if err != nil {
		log.Fatalf("Failed to new mcp server: %v", err)
	}

	if sse != "" {
		log.Printf("SSE MCP Server Starting")
		err = server.NewSSEServer(s).Start(sse)
	} else {
		err = server.ServeStdio(s)
	}
	if err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("Failed to serve MCP: %v", err)
	}
}
