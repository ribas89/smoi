package upstream

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"smoi/internal/storage"
)

type Client struct {
	Base         string
	Key          string
	DefaultModel string
	HTTP         *http.Client
	Store        *storage.Store
}

type Model struct {
	ID string `json:"id"`
}

func New(
	baseURL, apiKey, defaultModel string,
	timeout time.Duration, store *storage.Store,
) (*Client, error) {
	if baseURL == "" {
		return nil, errors.New("OPENAI_BASE_URL is required")
	}

	parsed, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil || parsed.Host == "" ||
		(parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, errors.New("invalid OPENAI_BASE_URL")
	}

	if !strings.HasSuffix(parsed.Path, "/v1") {
		parsed.Path = path.Join(parsed.Path, "v1")
	}
	if timeout < 0 {
		timeout = 0
	}

	return &Client{
		Base:         strings.TrimRight(parsed.String(), "/"),
		Key:          apiKey,
		DefaultModel: defaultModel,
		HTTP:         &http.Client{Timeout: timeout},
		Store:        store,
	}, nil
}

func (c *Client) request(
	ctx context.Context, method, endpoint string, body io.Reader,
) (*http.Request, error) {
	request, err := http.NewRequestWithContext(ctx, method, c.Base+endpoint, body)
	if err != nil {
		return nil, err
	}

	if c.Key != "" {
		request.Header.Set("Authorization", "Bearer "+c.Key)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	return request, nil
}

func (c *Client) Models(ctx context.Context) ([]Model, error) {
	request, err := c.request(ctx, http.MethodGet, "/models", nil)
	if err != nil {
		return nil, err
	}

	response, err := c.HTTP.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	slog.Debug("upstream models HTTP response", "status", response.StatusCode, "endpoint", request.URL.String())

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 2048))
		return nil, fmt.Errorf(
			"upstream models (%d): %s", response.StatusCode, strings.TrimSpace(string(body)),
		)
	}

	var result struct {
		Data []Model `json:"data"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&result); err != nil {
		return nil, err
	}
	if result.Data == nil {
		result.Data = []Model{}
	}

	slog.Debug("upstream models received", "models", result.Data)
	return result.Data, nil
}
