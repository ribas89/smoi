package upstream

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"smoi/internal/database"
)

type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content json.RawMessage `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Error json.RawMessage `json:"error"`
}

func (c *Client) Stream(
	ctx context.Context,
	model string,
	messages []database.Message,
	onDelta func(string) error,
) error {
	if model == "" {
		model = c.DefaultModel
	}
	if model == "" {
		return errors.New("no model selected; configure OPENAI_DEFAULT_MODEL or select a model")
	}

	reader, writer := io.Pipe()
	go func() {
		writer.CloseWithError(c.writePayload(writer, model, messages))
	}()
	defer reader.Close()

	request, err := c.request(ctx, http.MethodPost, "/chat/completions", reader)
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "text/event-stream")
	slog.Debug("upstream request sent", "method", request.Method, "endpoint", request.URL.String(),
		"model", model, "messages", len(messages))

	response, err := c.HTTP.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	slog.Debug("upstream HTTP response", "status", response.StatusCode, "endpoint", request.URL.String())

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		slog.Debug("upstream error body", "status", response.StatusCode, "body", string(body))
		return fmt.Errorf(
			"upstream returned HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)),
		)
	}

	return consumeStream(response.Body, onDelta)
}

func consumeStream(body io.Reader, onDelta func(string) error) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)

	var eventData []string
	completed := false

	flush := func() error {
		if len(eventData) == 0 {
			return nil
		}

		data := strings.Join(eventData, "\n")
		eventData = eventData[:0]
		if data == "[DONE]" {
			slog.Debug("upstream stream completed")
			completed = true
			return nil
		}

		return decodeStreamChunk(data, onDelta)
	}

	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")

		if line == "" {
			if err := flush(); err != nil {
				return err
			}
			if completed {
				return nil
			}
			continue
		}

		if strings.HasPrefix(line, "data:") {
			eventData = append(eventData, strings.TrimPrefix(
				strings.TrimPrefix(line, "data:"), " ",
			))
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	if err := flush(); err != nil {
		return err
	}
	if !completed {
		return errors.New("upstream stream ended without [DONE]")
	}
	return nil
}

func decodeStreamChunk(data string, onDelta func(string) error) error {
	slog.Debug("upstream SSE event", "payload", data)
	var chunk streamChunk
	if err := json.Unmarshal([]byte(data), &chunk); err != nil {
		return fmt.Errorf("invalid upstream event: %w", err)
	}
	if len(chunk.Error) > 0 && string(chunk.Error) != "null" {
		return fmt.Errorf("upstream error: %s", string(chunk.Error))
	}

	for _, choice := range chunk.Choices {
		var content string
		if json.Unmarshal(choice.Delta.Content, &content) != nil || content == "" {
			continue
		}
		if err := onDelta(content); err != nil {
			return err
		}
	}

	return nil
}
