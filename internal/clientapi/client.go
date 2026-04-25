package clientapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultHTTPTimeout = 10 * time.Second

// Client описывает HTTP-клиент для общения с сервером.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// APIError описывает ошибку, полученную от HTTP API.
type APIError struct {
	// StatusCode — HTTP-статус ответа.
	StatusCode int

	// Message — текст ошибки.
	Message string
}

// Error возвращает строковое представление API-ошибки.
func (e *APIError) Error() string {
	return fmt.Sprintf("api error: status=%d message=%q", e.StatusCode, e.Message)
}

type errorResponse struct {
	Error string `json:"error"`
}

// New создаёт новый HTTP-клиент для работы с сервером.
func New(baseURL string, httpClient *http.Client) (*Client, error) {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		return nil, fmt.Errorf("client api base url is empty")
	}

	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: defaultHTTPTimeout,
		}
	}

	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
	}, nil
}

func (c *Client) doJSON(
	ctx context.Context,
	method string,
	path string,
	requestBody any,
	responseBody any,
	headers map[string]string,
) error {
	var bodyReader io.Reader

	if requestBody != nil {
		payload, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}

		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("build http request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	if requestBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("do http request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= http.StatusBadRequest {
		return decodeAPIError(resp)
	}

	if responseBody == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(responseBody); err != nil {
		return fmt.Errorf("decode response body: %w", err)
	}

	return nil
}

func decodeAPIError(resp *http.Response) error {
	var apiErrBody errorResponse

	if err := json.NewDecoder(resp.Body).Decode(&apiErrBody); err != nil {
		return &APIError{
			StatusCode: resp.StatusCode,
			Message:    http.StatusText(resp.StatusCode),
		}
	}

	message := strings.TrimSpace(apiErrBody.Error)
	if message == "" {
		message = http.StatusText(resp.StatusCode)
	}

	return &APIError{
		StatusCode: resp.StatusCode,
		Message:    message,
	}
}