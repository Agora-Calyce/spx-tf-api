package spx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Client is the client for the SPX API
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	logger     zerolog.Logger
	token      string
}

// NewClient creates a new instance of the SPX client
func NewClient(baseURLStr string, token string) (*Client, error) {
	baseURL, err := url.Parse(baseURLStr)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logger:     log.Logger,
		token:      token,
	}, nil
}

// Do executes an HTTP request
func (c *Client) do(ctx context.Context, method, path string, body interface{}, params url.Values) (*http.Response, error) {
	// JoinPath keeps the endpoint path (e.g. "https://host/api");
	// Parse would replace this path because path starts with "/".
	u := c.baseURL.JoinPath(path)
	if params != nil {
		u.RawQuery = params.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBody)

		// Log the request body in debug mode (formatted)
		if c.logger.GetLevel() <= zerolog.DebugLevel {
			formattedBody, err := json.MarshalIndent(body, "", "  ")
			if err == nil {
				c.logger.Debug().
					Str("method", method).
					Str("url", u.String()).
					Str("body", string(formattedBody)).
					Msg("Client API request body")
			}
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "api-token "+c.token)
	}

	c.logger.Info().
		Str("method", method).
		Str("url", u.String()).
		Msg("Client API call")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error().
			Err(err).
			Str("method", method).
			Str("url", u.String()).
			Msg("Client API call failed")
		return nil, err
	}

	// Log the body in debug mode
	if c.logger.GetLevel() <= zerolog.DebugLevel {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err == nil {
			c.logger.Debug().
				Str("method", method).
				Str("url", u.String()).
				Int("status_code", resp.StatusCode).
				Str("body", string(bodyBytes)).
				Msg("Client API response body")
			// Restore the body for what follows
			resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
	}

	c.logger.Info().
		Str("method", method).
		Str("url", u.String()).
		Int("status_code", resp.StatusCode).
		Msg("Client API response")

	return resp, nil
}

// APIError is the error returned when the SPX API responds with a status >= 400
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Body)
}

// StatusCode returns the HTTP status of the SPX response held in err, or 0 if err holds none
func StatusCode(err error) int {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode
	}
	return 0
}

func decodeResponse(resp *http.Response, v interface{}) error {
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return &APIError{StatusCode: resp.StatusCode, Body: string(body)}
	}
	if v == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(v)
}
