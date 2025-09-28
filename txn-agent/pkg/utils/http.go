package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPClientImpl implements the HTTPClient interface
type HTTPClientImpl struct {
	client  *http.Client
	timeout time.Duration
}

// NewHTTPClient creates a new HTTP client with the specified timeout
func NewHTTPClient(timeout time.Duration) *HTTPClientImpl {
	return &HTTPClientImpl{
		client: &http.Client{
			Timeout: timeout,
		},
		timeout: timeout,
	}
}

// Post performs a POST request and returns the response body
func (h *HTTPClientImpl) Post(ctx context.Context, url string, headers map[string]string, body interface{}) ([]byte, error) {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return respBody, nil
}

// PostStream performs a POST request and returns a stream for reading the response
func (h *HTTPClientImpl) PostStream(ctx context.Context, url string, headers map[string]string, body interface{}) (io.ReadCloser, error) {
	var reqBody []byte
	var err error

	if body != nil {
		reqBody, err = json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return resp.Body, nil
}

// RetryHTTPClient wraps an HTTP client with retry logic
type RetryHTTPClient struct {
	client     *HTTPClientImpl
	maxRetries int
	retryDelay time.Duration
}

// NewRetryHTTPClient creates a new HTTP client with retry logic
func NewRetryHTTPClient(timeout time.Duration, maxRetries int, retryDelay time.Duration) *RetryHTTPClient {
	return &RetryHTTPClient{
		client:     NewHTTPClient(timeout),
		maxRetries: maxRetries,
		retryDelay: retryDelay,
	}
}

// Post performs a POST request with retry logic
func (r *RetryHTTPClient) Post(ctx context.Context, url string, headers map[string]string, body interface{}) ([]byte, error) {
	var lastErr error

	for i := 0; i <= r.maxRetries; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(r.retryDelay):
			}
		}

		resp, err := r.client.Post(ctx, url, headers, body)
		if err == nil {
			return resp, nil
		}

		lastErr = err

		if !isRetryableError(err) {
			break
		}
	}

	return nil, fmt.Errorf("request failed after %d retries: %w", r.maxRetries, lastErr)
}

// PostStream performs a POST request with retry logic for streaming
func (r *RetryHTTPClient) PostStream(ctx context.Context, url string, headers map[string]string, body interface{}) (io.ReadCloser, error) {
	var lastErr error

	for i := 0; i <= r.maxRetries; i++ {
		if i > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(r.retryDelay):
			}
		}

		stream, err := r.client.PostStream(ctx, url, headers, body)
		if err == nil {
			return stream, nil
		}

		lastErr = err

		if !isRetryableError(err) {
			break
		}
	}

	return nil, fmt.Errorf("streaming request failed after %d retries: %w", r.maxRetries, lastErr)
}

// isRetryableError determines if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Check for network errors, timeouts, and certain HTTP status codes
	errStr := err.Error()
	return bytes.Contains([]byte(errStr), []byte("timeout")) ||
		bytes.Contains([]byte(errStr), []byte("connection refused")) ||
		bytes.Contains([]byte(errStr), []byte("HTTP error 429")) ||
		bytes.Contains([]byte(errStr), []byte("HTTP error 5"))
}
