package common_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/klauspost/compress/zstd"
	"github.com/labring/aiproxy/core/common"
)

func TestGetRequestBodyReusableJSONWithContentLength(t *testing.T) {
	req := httptest.NewRequestWithContext(
		context.Background(),
		"POST",
		"/v1/chat/completions",
		strings.NewReader(`{"a":1}`),
	)
	req.Header.Set("Content-Type", "application/json")

	body, err := common.GetRequestBodyReusable(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(body) != `{"a":1}` {
		t.Fatalf("unexpected body: %q", string(body))
	}

	cached, ok := common.GetCachedRequestBody(req)
	if !ok {
		t.Fatal("expected cached request body")
	}

	if string(cached) != `{"a":1}` {
		t.Fatalf("unexpected cached body: %q", string(cached))
	}
}

func TestSetRequestBodySyncsBodyAndContentLength(t *testing.T) {
	req := httptest.NewRequestWithContext(
		context.Background(),
		"POST",
		"/v1/chat/completions",
		strings.NewReader(`{"a":1}`),
	)

	common.SetRequestBody(req, []byte(`{"b":2}`))

	if req.ContentLength != int64(len(`{"b":2}`)) {
		t.Fatalf("unexpected content length: %d", req.ContentLength)
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("unexpected read error: %v", err)
	}

	if string(body) != `{"b":2}` {
		t.Fatalf("unexpected body: %q", string(body))
	}

	if req.GetBody != nil {
		t.Fatal("expected GetBody to be cleared")
	}
}

func TestGetRequestBodyReusableDecodesZstd(t *testing.T) {
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		t.Fatalf("unexpected encoder error: %v", err)
	}
	defer encoder.Close()

	original := []byte(`{"model":"gpt-5.6-sol","input":[{"type":"compaction_trigger"}]}`)
	compressed := encoder.EncodeAll(original, nil)
	req := httptest.NewRequestWithContext(
		context.Background(),
		"POST",
		"/v1/responses",
		bytes.NewReader(compressed),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "zstd")

	body, err := common.GetRequestBodyReusable(req)
	if err != nil {
		t.Fatalf("unexpected decode error: %v", err)
	}

	if !bytes.Equal(body, original) {
		t.Fatalf("unexpected decoded body: %q", body)
	}

	if got := req.Header.Get("Content-Encoding"); got != "" {
		t.Fatalf("expected content encoding to be cleared, got %q", got)
	}

	if req.ContentLength != int64(len(original)) {
		t.Fatalf("unexpected content length: %d", req.ContentLength)
	}
}

func TestGetRequestBodyReusableRejectsOversizedZstdOutput(t *testing.T) {
	encoder, err := zstd.NewWriter(nil)
	if err != nil {
		t.Fatalf("unexpected encoder error: %v", err)
	}
	defer encoder.Close()

	compressed := encoder.EncodeAll(bytes.Repeat([]byte{'a'}, common.MaxRequestBodySize+1), nil)
	req := httptest.NewRequestWithContext(
		context.Background(),
		"POST",
		"/v1/responses",
		bytes.NewReader(compressed),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "zstd")

	_, err = common.GetRequestBodyReusable(req)
	if err == nil || !strings.Contains(err.Error(), "decompressed request body too large") {
		t.Fatalf("expected decompressed size error, got %v", err)
	}
}

func TestGetRequestBodyReusableRejectsCorruptZstd(t *testing.T) {
	req := httptest.NewRequestWithContext(
		context.Background(),
		"POST",
		"/v1/responses",
		strings.NewReader("invalid zstd data"),
	)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "zstd")

	_, err := common.GetRequestBodyReusable(req)
	if err == nil || !strings.Contains(err.Error(), "zstd request body decode failed") {
		t.Fatalf("expected zstd decode error, got %v", err)
	}
}

func TestParseFormWithLimitRejectsTooLargeContentLength(t *testing.T) {
	req := httptest.NewRequestWithContext(
		context.Background(),
		"POST",
		"/v1/images/edits",
		strings.NewReader("n=1"),
	)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.ContentLength = common.MaxRequestBodySize + 1

	err := common.ParseFormWithLimit(req)
	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "request body too large") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetJSONNodeNoCopy(t *testing.T) {
	node, err := common.GetJSONNodeNoCopy(
		[]byte(`{"choices":[{"message":{"content":"hello"}}]}`),
		"choices",
		0,
		"message",
		"content",
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	value, err := node.String()
	if err != nil {
		t.Fatalf("unexpected string error: %v", err)
	}

	if value != "hello" {
		t.Fatalf("unexpected value: %q", value)
	}
}

func TestGetResponseBodyLimitKnownLengthTooLargeHidesLimit(t *testing.T) {
	resp := &http.Response{
		Body:          io.NopCloser(strings.NewReader("abcd")),
		ContentLength: 4,
	}

	_, err := common.GetResponseBodyLimit(resp, 3)
	if err == nil {
		t.Fatal("expected error")
	}

	if got := err.Error(); got != "response body too large" {
		t.Fatalf("unexpected error: %q", got)
	}
}

func TestGetResponseBodyLimitUnknownLengthTooLargeHidesLimit(t *testing.T) {
	resp := &http.Response{
		Body:          io.NopCloser(strings.NewReader("abcd")),
		ContentLength: -1,
	}

	_, err := common.GetResponseBodyLimit(resp, 3)
	if err == nil {
		t.Fatal("expected error")
	}

	if got := err.Error(); got != "response body too large" {
		t.Fatalf("unexpected error: %q", got)
	}
}
