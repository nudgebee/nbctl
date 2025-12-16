package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"os"
	"sync"
	"time"

	"github.com/machinebox/graphql"
	"github.com/spf13/viper"
	"nudgebee.com/nbctl/pkg/config"
)

// loggingTransport is an http.RoundTripper that logs requests and responses.
type loggingTransport struct {
	wrapped http.RoundTripper
	logger  *log.Logger
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Log the request
	reqDump, err := httputil.DumpRequestOut(req, true)
	if err != nil {
		t.logger.Printf("Error dumping request: %v", err)
	} else {
		t.logger.Printf("Request:\n%s", reqDump)
	}

	resp, err := t.wrapped.RoundTrip(req)
	if err != nil {
		t.logger.Printf("Error sending request: %v", err)
		return nil, err
	}

	// Log the response
	// We need to read the body and then replace it.
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		t.logger.Printf("Error reading response body: %v", readErr)
		return resp, err // return original response and error
	}
	if err := resp.Body.Close(); err != nil {
		t.logger.Printf("Error closing response body: %v", err)
	} // close original body

	// Create a new response with the same body, so it can be read again.
	resp.Body = io.NopCloser(bytes.NewBuffer(body))

	// Dump the response for logging.
	respDump, dumpErr := httputil.DumpResponse(resp, true)
	if dumpErr != nil {
		t.logger.Printf("Error dumping response: %v", dumpErr)
	} else {
		t.logger.Printf("Response:\n%s", respDump)
	}

	// Restore the original body so it can be read by the caller.
	resp.Body = io.NopCloser(bytes.NewBuffer(body))

	return resp, err
}

type clientOptions struct {
	endpoint string
	apiKey   string
	username string
}

type ClientOption interface {
	apply(opts *clientOptions)
}

type clientUsernameOption struct {
	username string
}

func (o clientUsernameOption) apply(opts *clientOptions) {
	if o.username != "" {
		opts.username = o.username
	}
}

func WithUsername(username string) ClientOption {
	return clientUsernameOption{
		username: username,
	}
}

type clientApiKeyOption struct {
	apiKey string
}

func (o clientApiKeyOption) apply(opts *clientOptions) {
	if o.apiKey != "" {
		opts.apiKey = o.apiKey
	}
}

func WithApiKey(apiKey string) ClientOption {
	return clientApiKeyOption{
		apiKey: apiKey,
	}
}

type clientEndpointOption struct {
	endpoint string
}

func (o clientEndpointOption) apply(opts *clientOptions) {
	if o.endpoint != "" {
		opts.endpoint = o.endpoint
	}
}

func WithEndpoint(endpoint string) ClientOption {
	return clientEndpointOption{
		endpoint: endpoint,
	}
}

// NewClient creates a new GraphQL client.
func NewClient(opts ...ClientOption) *graphql.Client {
	config := clientOptions{}
	for _, o := range opts {
		o.apply(&config)
	}

	endpoint := config.endpoint
	if endpoint == "" {
		endpoint = viper.GetString("endpoint")
	}
	if endpoint == "" {
		endpoint = "https://app.nudgebee.com"
	}
	if endpoint[len(endpoint)-1] == '/' {
		endpoint = endpoint[:len(endpoint)-1]
	}
	graphqlEndpoint := endpoint + "/api/graphql"

	apiKey := config.apiKey
	if apiKey == "" {
		apiKey = viper.GetString("api-key")
	}

	username := config.username
	if username == "" {
		username = viper.GetString("username")
	}
	tokenEndpoint := endpoint + "/api/auth/token"

	// create a new http client with the auth header
	// transport that injects bearer tokens obtained from token endpoint
	transport := &authTransport{
		apiKey:        apiKey,
		username:      username,
		tokenEndpoint: tokenEndpoint,
		wrapped:       http.DefaultTransport,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
	}

	var finalTransport http.RoundTripper = transport
	verbose := viper.GetBool("verbose")
	if verbose {
		logFile, err := os.OpenFile("nbctl_graphql.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			log.Printf("Error opening log file: %v\n", err)
		} else {
			logger := log.New(logFile, "", log.LstdFlags)
			finalTransport = &loggingTransport{
				wrapped: transport,
				logger:  logger,
			}
		}
	}

	httpClient := &http.Client{
		Transport: finalTransport,
	}

	client := graphql.NewClient(graphqlEndpoint, graphql.WithHTTPClient(httpClient))

	return client
}

type authTransport struct {
	apiKey        string
	username      string
	tokenEndpoint string
	wrapped       http.RoundTripper

	// http client used to fetch tokens
	httpClient *http.Client

	// cached token state
	mu          sync.Mutex
	accessToken string
	expiry      time.Time
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// ensure we have a valid access token
	token, err := t.getAccessToken(req.Context())
	if err != nil {
		return nil, err
	}

	// avoid mutating original request
	req2 := cloneRequest(req)
	req2.Header.Set("Authorization", "Bearer "+token)

	resp, err := t.wrapped.RoundTrip(req2)
	if err != nil {
		return resp, err
	}

	// if unauthorized, try refreshing token and retry once
	if resp.StatusCode == http.StatusUnauthorized {
		// discard body from first response; handle potential copy error
		if _, err := io.Copy(io.Discard, resp.Body); err != nil {
			_ = err // intentionally ignore copy error
		}
		if err := resp.Body.Close(); err != nil {
			_ = err // intentionally ignore close error
		}

		// force refresh
		if err := t.forceRefresh(req.Context()); err != nil {
			return nil, err
		}

		// retry request with new token
		token, err = t.getAccessToken(req.Context())
		if err != nil {
			return nil, err
		}

		req3 := cloneRequest(req)
		req3.Header.Set("Authorization", "Bearer "+token)
		return t.wrapped.RoundTrip(req3)
	}

	return resp, nil
}

// cloneRequest creates a shallow copy of the request with a deep copy of the Header
func cloneRequest(r *http.Request) *http.Request {
	r2 := r.Clone(r.Context())
	// Clone doesn't deep-copy Header map, so make a copy
	hdr := make(http.Header, len(r.Header))
	for k, vv := range r.Header {
		vv2 := make([]string, len(vv))
		copy(vv2, vv)
		hdr[k] = vv2
	}
	r2.Header = hdr
	return r2
}

// tokenResponse models the expected JSON response from the token endpoint.
// We accept multiple common field names.
type tokenResponse struct {
	Token  string `json:"token"`
	Expiry int64  `json:"expiry"`
}

// getAccessToken returns a valid access token, fetching a new one if needed.
func (t *authTransport) getAccessToken(ctx context.Context) (string, error) {
	t.mu.Lock()
	token := t.accessToken
	exp := t.expiry
	t.mu.Unlock()

	if token != "" && time.Now().Before(exp) {
		return token, nil
	}

	// fetch new token
	if err := t.fetchToken(ctx); err != nil {
		return "", err
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	return t.accessToken, nil
}

// forceRefresh forces fetching a new token regardless of cached expiry
func (t *authTransport) forceRefresh(ctx context.Context) error {
	return t.fetchToken(ctx)
}

// fetchToken calls the token endpoint with the API key and stores the access token and expiry
func (t *authTransport) fetchToken(ctx context.Context) error {
	// prepare request. We POST an empty JSON body and include the api-key in header 'X-Api-Key'.
	// NOTE: This is an assumption; if your token endpoint expects a different shape (e.g. JSON body),
	// adjust accordingly or set a custom token-endpoint implementation.
	bodyBytes, err := json.Marshal(map[string]string{
		"email":  t.username,
		"secret": t.apiKey,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.tokenEndpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.New("token endpoint returned non-2xx status: " + resp.Status)
	}

	var tr tokenResponse
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&tr); err != nil {
		return err
	}

	token := tr.Token
	if token == "" {
		return errors.New("token endpoint did not return access_token")
	}

	// calculate expiry
	var expiry time.Time
	if tr.Expiry > 0 {
		// subtract small buffer
		expiry = time.Now().Add(time.Duration(tr.Expiry)*time.Second - 10*time.Second)
	} else {
		// default to 55 minutes
		expiry = time.Now().Add(55 * time.Minute)
	}

	t.mu.Lock()
	t.accessToken = token
	t.expiry = expiry
	t.mu.Unlock()

	return nil
}

type GraphQLError struct {
	Message    string          `json:"message"`
	Extensions json.RawMessage `json:"extensions"`
}

type GraphQLErrors []GraphQLError

func (e GraphQLErrors) Error() string {
	if len(e) == 0 {
		return ""
	}
	var buf bytes.Buffer
	for i, err := range e {
		if i > 0 {
			buf.WriteString("; ")
		}
		buf.WriteString(err.Message)
	}
	return buf.String()
}

var (
	runClient     *graphql.Client
	runClientOnce sync.Once
)

// getRunClient returns the singleton GraphQL client, initializing it if necessary.
func getRunClient() *graphql.Client {
	runClientOnce.Do(func() {
		runClient = NewClient()
	})
	return runClient
}

// ResetClient resets the shared client instance. This is primarily for testing.
// Warning: This function is not thread-safe and should only be used in test cleanup.
func ResetClient() {
	runClient = nil
	runClientOnce = sync.Once{}
}

// Run executes a GraphQL request.
func Run(ctx context.Context, req *graphql.Request, resp any) error {
	config.InitConfig()
	client := getRunClient()

	verbose := viper.GetBool("verbose")
	if verbose {
		logFile, err := os.OpenFile("nbctl_graphql.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			log.Printf("Error opening log file: %v\n", err)
		} else {
			defer func() {
				if err := logFile.Close(); err != nil {
					slog.Error("unable to close log file", "error", err)
				}
			}()
			logger := log.New(logFile, "", log.LstdFlags)

			// Log the GraphQL request
			requestBody, err := json.MarshalIndent(req, "", "  ")
			if err != nil {
				logger.Printf("Error marshalling GraphQL request: %v\n", err)
			} else {
				logger.Printf("GraphQL Request: %s\n", requestBody)
			}
		}
	}

	// Execute the GraphQL request into a temporary struct that captures both data and errors.
	var rawGraphQLResponse struct {
		Data   json.RawMessage `json:"data"`
		Errors []GraphQLError  `json:"errors"`
	}

	// Use the underlying machinebox/graphql client's Run method.
	// This will handle network errors and HTTP status codes.
	err := client.Run(ctx, req, &rawGraphQLResponse)

	if err != nil {
		// This error is from the underlying HTTP request or client.Run itself (e.g., non-2xx/4xx status, network error).
		return fmt.Errorf("GraphQL client execution failed: %w", err)
	}

	// If the GraphQL response contains errors, return our custom error type.
	if len(rawGraphQLResponse.Errors) > 0 {
		return GraphQLErrors(rawGraphQLResponse.Errors)
	}

	// If no GraphQL errors, unmarshal the data into the caller's response struct.
	if err := json.Unmarshal(rawGraphQLResponse.Data, resp); err != nil {
		return fmt.Errorf("failed to unmarshal GraphQL data: %w", err)
	}

	if verbose {
		logFile, fileErr := os.OpenFile("nbctl_graphql.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if fileErr != nil {
			log.Printf("Error opening log file: %v\n", fileErr)
		} else {
			defer func() {
				if err := logFile.Close(); err != nil {
					slog.Error("unable to close log file", "error", err)
				}
			}()
			logger := log.New(logFile, "", log.LstdFlags)

			// Log the GraphQL response or error
			if err != nil {
				logger.Printf("GraphQL Response Error: %v\n", err)
			} else {
				respBytes, marshalErr := json.MarshalIndent(resp, "", "  ")
				if marshalErr != nil {
					logger.Printf("Error marshalling GraphQL response: %v\n", marshalErr)
				} else {
					logger.Printf("GraphQL Response: %s\n", respBytes)
				}
			}
		}
	}

	return nil
}
