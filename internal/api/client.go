// Package api provides the HTTPS client for the Bluff service.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
)

const maxResponseBytes = 2 << 20

// HTTPClient is implemented by http.Client and makes the API client testable.
type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

// Client calls the Bluff API.
type Client struct {
	baseURL *url.URL
	http    HTTPClient
}

// Health verifies that the Bluff service is reachable.
func (c *Client) Health(ctx context.Context) error {
	result, err := request[struct {
		Status string `json:"status"`
	}](ctx, c, http.MethodGet, "/health", "", nil)
	if err != nil {
		return err
	}
	if result.Status != "ok" {
		return fmt.Errorf("bluff service reported status %q", result.Status)
	}
	return nil
}

// Error is a problem returned by the Bluff API.
type Error struct {
	Status    int
	Code      string
	Message   string
	RequestID string
}

func (e *Error) Error() string {
	if e.RequestID == "" {
		return e.Message
	}
	return fmt.Sprintf("%s (request %s)", e.Message, e.RequestID)
}

// IsUnauthorized reports whether an error means the session is invalid.
func IsUnauthorized(err error) bool {
	var apiErr *Error
	return errors.As(err, &apiErr) && apiErr.Status == http.StatusUnauthorized
}

// NewClient constructs an API client for an HTTPS endpoint. HTTP is accepted
// only for localhost development.
func NewClient(rawBaseURL string, httpClient HTTPClient) (*Client, error) {
	parsed, err := url.Parse(strings.TrimRight(rawBaseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}
	secureRemote := parsed.Scheme == "https"
	localDevelopment := parsed.Scheme == "http" && isLoopback(parsed.Hostname())
	if parsed.Host == "" || (!secureRemote && !localDevelopment) {
		return nil, errors.New("API URL must use HTTPS, except for localhost development")
	}
	if httpClient == nil {
		return nil, errors.New("HTTP client is required")
	}
	return &Client{baseURL: parsed, http: httpClient}, nil
}

func isLoopback(host string) bool {
	return host == "localhost" || net.ParseIP(host).IsLoopback()
}

// Login exchanges credentials for a server session.
func (c *Client) Login(ctx context.Context, username, password string) (Session, error) {
	return request[Session](ctx, c, http.MethodPost, "/v1/auth/login", "", map[string]string{
		"username": username,
		"password": password,
	})
}

// ValidateInvitation verifies a code before account details are requested.
func (c *Client) ValidateInvitation(ctx context.Context, code string) error {
	_, err := request[struct {
		Valid bool `json:"valid"`
	}](ctx, c, http.MethodPost, "/v1/auth/invitations/validate", "", map[string]string{"code": code})
	return err
}

// RedeemInvitation creates a member account and returns its authenticated session.
func (c *Client) RedeemInvitation(ctx context.Context, code, username, password string) (Session, error) {
	return request[Session](ctx, c, http.MethodPost, "/v1/auth/invitations/redeem", "", map[string]string{
		"code": code, "username": username, "password": password,
	})
}

// Me returns the account attached to a session token.
func (c *Client) Me(ctx context.Context, token string) (User, error) {
	result, err := request[struct {
		User User `json:"user"`
	}](ctx, c, http.MethodGet, "/v1/auth/me", token, nil)
	return result.User, err
}

// Users returns every active Bluff account. The service restricts this to administrators.
func (c *Client) Users(ctx context.Context, token string) ([]User, error) {
	result, err := request[struct {
		Users []User `json:"users"`
	}](ctx, c, http.MethodGet, "/v1/auth/users", token, nil)
	return result.Users, err
}

// CreateInvitation creates a single-use invite. The service restricts this to administrators.
func (c *Client) CreateInvitation(ctx context.Context, token string) (Invitation, error) {
	result, err := request[struct {
		Invitation Invitation `json:"invitation"`
	}](ctx, c, http.MethodPost, "/v1/auth/invitations", token, struct{}{})
	return result.Invitation, err
}

// Bootstrap returns the complete dashboard state.
func (c *Client) Bootstrap(ctx context.Context, token string) (Bootstrap, error) {
	return request[Bootstrap](ctx, c, http.MethodGet, "/v1/bootstrap", token, nil)
}

// Logout revokes the current server session.
func (c *Client) Logout(ctx context.Context, token string) error {
	_, err := request[struct {
		LoggedOut bool `json:"loggedOut"`
	}](ctx, c, http.MethodPost, "/v1/auth/logout", token, struct{}{})
	return err
}

type envelope[T any] struct {
	OK        bool       `json:"ok"`
	Data      T          `json:"data"`
	Error     apiProblem `json:"error"`
	RequestID string     `json:"requestId"`
}

type apiProblem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func request[T any](ctx context.Context, client *Client, method, path, token string, body any) (T, error) {
	var zero T
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return zero, fmt.Errorf("encode request: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}

	endpoint := client.baseURL.ResolveReference(&url.URL{Path: path})
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), reader)
	if err != nil {
		return zero, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	response, err := client.http.Do(req)
	if err != nil {
		return zero, fmt.Errorf("contact Bluff: %w", err)
	}
	defer func() { _ = response.Body.Close() }()

	decoded := envelope[T]{}
	limited := io.LimitReader(response.Body, maxResponseBytes)
	if err := json.NewDecoder(limited).Decode(&decoded); err != nil {
		return zero, fmt.Errorf("decode Bluff response: %w", err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 || !decoded.OK {
		message := decoded.Error.Message
		if message == "" {
			message = http.StatusText(response.StatusCode)
		}
		return zero, &Error{Status: response.StatusCode, Code: decoded.Error.Code, Message: message, RequestID: decoded.RequestID}
	}
	return decoded.Data, nil
}
