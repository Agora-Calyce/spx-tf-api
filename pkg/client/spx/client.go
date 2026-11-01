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

// Client est le client pour l'API SPX
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
	logger     zerolog.Logger
	token      string
}

// NewClient crée une nouvelle instance du client SPX
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

// Do execute une requête HTTP
func (c *Client) do(ctx context.Context, method, path string, body interface{}, params url.Values) (*http.Response, error) {
	// JoinPath conserve le chemin de l'endpoint (ex. "https://host/api") ;
	// Parse remplacerait ce chemin car path commence par "/".
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

		// Logging du body de la requête en mode debug (formatté)
		if c.logger.GetLevel() <= zerolog.DebugLevel {
			formattedBody, err := json.MarshalIndent(body, "", "  ")
			if err == nil {
				c.logger.Debug().
					Str("method", method).
					Str("url", u.String()).
					Str("body", string(formattedBody)).
					Msg("Body de la requête API cliente")
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
		Msg("Appel API cliente")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error().
			Err(err).
			Str("method", method).
			Str("url", u.String()).
			Msg("Échec de l'appel API cliente")
		return nil, err
	}

	// Logging du body en mode debug
	if c.logger.GetLevel() <= zerolog.DebugLevel {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err == nil {
			c.logger.Debug().
				Str("method", method).
				Str("url", u.String()).
				Int("status_code", resp.StatusCode).
				Str("body", string(bodyBytes)).
				Msg("Body de la réponse API cliente")
			// On remet le body pour la suite
			resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
	}

	c.logger.Info().
		Str("method", method).
		Str("url", u.String()).
		Int("status_code", resp.StatusCode).
		Msg("Réponse API cliente")

	return resp, nil
}

// APIError est l'erreur renvoyée quand l'API SPX répond avec un statut >= 400
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Body)
}

// StatusCode renvoie le statut HTTP de la réponse SPX contenue dans err, ou 0 si err n'en contient pas
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
