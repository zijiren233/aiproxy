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

func main() {
	flag.Parse()

	if file == "" {
		log.Fatal("Not provied openapi file")
	}

	parser := convert.NewParser()
	var err error

	if strings.HasPrefix(file, "http://") || strings.HasPrefix(file, "https://") {
		// For HTTP URLs, fetch the content first
		var resp *http.Response
		resp, err = http.Get(file)
		if err != nil {
			log.Fatalf("Failed to fetch OpenAPI document from URL: %v", err)
		}
		defer resp.Body.Close()

		var content []byte
		content, err = io.ReadAll(resp.Body)
		if err != nil {
			log.Fatalf("Failed to read OpenAPI document from response: %v", err)
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
		log.Fatalf("Failed to parse OpenAPI document: %v", err)
	}
	converter := convert.NewConverter(parser, convert.Options{})
	s, err := converter.Convert()
	if err != nil {
		log.Fatalf("Failed to convert OpenAPI to MCP: %v", err)
	}

	if sse != "" {
		err = server.NewSSEServer(s).Start(sse)
	} else {
		err = server.ServeStdio(s)
	}
	if err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("Failed to serve MCP: %v", err)
	}
}
