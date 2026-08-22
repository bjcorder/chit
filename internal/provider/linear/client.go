package linear

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

const defaultEndpoint = "https://api.linear.app/graphql"

// graphQLClient is a minimal GraphQL client for Linear's single-endpoint
// API. There's no official or widely-adopted Go SDK for Linear (see
// docs/research/linear-api.md), and chit's query surface is small enough
// that hand-rolling this is simpler than adopting a generic GraphQL client
// library just for a handful of queries and one mutation.
type graphQLClient struct {
	httpClient *http.Client
	endpoint   string
	apiKey     string
}

func newGraphQLClient(apiKey string) *graphQLClient {
	return &graphQLClient{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		endpoint:   defaultEndpoint,
		apiKey:     apiKey,
	}
}

type graphQLError struct {
	Message string `json:"message"`
}

// do sends query with variables and decodes the "data" field into out.
// Variables are sent via GraphQL's own $var mechanism, never interpolated
// into the query string, so there's no injection risk from issue bodies,
// comment text, or IDs.
func (c *graphQLClient) do(ctx context.Context, query string, variables map[string]any, out any) error {
	reqBody, err := json.Marshal(map[string]any{"query": query, "variables": variables})
	if err != nil {
		return fmt.Errorf("linear: encoding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("linear: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.apiKey) // personal API keys use a bare token, no "Bearer" prefix

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("linear: request failed: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("linear: reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("linear: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors []graphQLError  `json:"errors"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("linear: parsing response: %w", err)
	}
	if len(envelope.Errors) > 0 {
		msgs := make([]string, len(envelope.Errors))
		for i, e := range envelope.Errors {
			msgs[i] = e.Message
		}
		return fmt.Errorf("linear: %s", strings.Join(msgs, "; "))
	}

	if out != nil && len(envelope.Data) > 0 {
		if err := json.Unmarshal(envelope.Data, out); err != nil {
			return fmt.Errorf("linear: parsing data: %w", err)
		}
	}
	return nil
}
