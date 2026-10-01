package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/nudgebee/nbctl/pkg/config"
	"github.com/spf13/viper"
)

// Request is a GraphQL request.
type Request struct {
	q      string
	vars   map[string]interface{}
	header http.Header
}

// NewRequest creates a new GraphQL request.
func NewRequest(q string) *Request {
	return &Request{
		q:      q,
		vars:   make(map[string]interface{}),
		header: make(http.Header),
	}
}

// Var sets a variable.
func (r *Request) Var(key string, value interface{}) {
	r.vars[key] = value
}

// Header sets a header.
func (r *Request) Header(key, value string) {
	r.header.Set(key, value)
}

// Client is a GraphQL client.
type Client struct {
	endpoint   string
	apiKey     string
	httpClient *http.Client
}

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
func NewClient(opts ...ClientOption) *Client {
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

	// create a new http client that sends the API key as the bearer token,
	// falling back to the token endpoint on servers that predate that
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
			// Wraps the auth transport, so it logs each request before the
			// Authorization header is added and the API key never reaches the file.
			finalTransport = &loggingTransport{
				wrapped: transport,
				logger:  logger,
			}
		}
	}

	httpClient := &http.Client{
		Transport: finalTransport,
		Timeout:   30 * time.Second,
	}

	return &Client{
		endpoint:   graphqlEndpoint,
		apiKey:     apiKey,
		httpClient: httpClient,
	}
}

// NewHTTPClient creates a new authenticated http.Client.
func NewHTTPClient(opts ...ClientOption) *http.Client {
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

	apiKey := config.apiKey
	if apiKey == "" {
		apiKey = viper.GetString("api-key")
	}

	username := config.username
	if username == "" {
		username = viper.GetString("username")
	}
	tokenEndpoint := endpoint + "/api/auth/token"

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

	return &http.Client{
		Transport: finalTransport,
		Timeout:   30 * time.Second,
	}
}

// apiKeyPrefix and gatewayKeyPrefix are the formats Nudgebee issues API keys
// in. A current server accepts a platform key (sk-nb-…) directly as the bearer
// token; an AI Gateway key (sk-nb-gw-…) only works against the AI Gateway.
const (
	apiKeyPrefix     = "sk-nb-"
	gatewayKeyPrefix = "sk-nb-gw-"
)

// authTransport sends the API key itself as the bearer token. A server that
// predates direct API-key auth refuses that with a 401 but still exchanges the
// key for a session token at tokenEndpoint, so on the first refusal the
// transport tries the exchange and, if it works, uses exchanged tokens from
// then on. Drop the exchange once no supported server predates direct auth.
type authTransport struct {
	apiKey        string
	username      string
	tokenEndpoint string
	wrapped       http.RoundTripper

	// http client used to fetch tokens
	httpClient *http.Client

	// cached token state
	mu          sync.Mutex
	legacy      bool
	accessToken string
	expiry      time.Time
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.mu.Lock()
	legacy := t.legacy
	t.mu.Unlock()
	if legacy {
		return t.roundTripExchanged(req)
	}

	req2, err := replayableRequest(req)
	if err != nil {
		return nil, err
	}
	req2.Header.Set("Authorization", "Bearer "+t.apiKey)
	resp, err := t.wrapped.RoundTrip(req2)
	if err != nil || resp.StatusCode != http.StatusUnauthorized {
		return resp, err
	}

	// Refused: either the key is bad, or the server predates direct API-key
	// auth. Only the exchange can tell the two apart. If it fails too, the
	// original 401 stands.
	if err := t.fetchToken(req.Context()); err != nil {
		return resp, nil
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()

	t.mu.Lock()
	t.legacy = true
	t.mu.Unlock()
	return t.roundTripExchanged(req)
}

// roundTripExchanged authenticates with a session token obtained from the
// token endpoint, for servers that predate direct API-key auth.
func (t *authTransport) roundTripExchanged(req *http.Request) (*http.Response, error) {
	// ensure we have a valid access token
	token, err := t.getAccessToken(req.Context())
	if err != nil {
		return nil, err
	}

	// avoid mutating original request
	req2, err := replayableRequest(req)
	if err != nil {
		return nil, err
	}
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

		req3, err := replayableRequest(req)
		if err != nil {
			return nil, err
		}
		req3.Header.Set("Authorization", "Bearer "+token)
		return t.wrapped.RoundTrip(req3)
	}

	return resp, nil
}

// replayableRequest clones req with a body of its own. A request can be sent
// more than once here (after a 401), and a plain clone shares the body reader
// an earlier attempt may already have drained, which fails with
// "ContentLength=N with Body length 0".
func replayableRequest(req *http.Request) (*http.Request, error) {
	r := cloneRequest(req)
	if req.GetBody != nil {
		body, err := req.GetBody()
		if err != nil {
			return nil, err
		}
		r.Body = body
	}
	return r, nil
}

// cloneRequest creates a deep copy of the request, including the Header
func cloneRequest(r *http.Request) *http.Request {
	// Clone returns a deep copy of r with its context changed to ctx.
	// The Request.Header map is also deep copied.
	return r.Clone(r.Context())
}

// unauthorizedError explains a 401. The app's 401 body carries neither `data`
// nor `errors`, so without this a rejected key would look like an empty result.
func unauthorizedError(apiKey string) error {
	switch {
	case apiKey == "":
		return errors.New("not authenticated: no API key configured; run `nbctl configure`")
	case strings.HasPrefix(apiKey, gatewayKeyPrefix):
		return errors.New("not authenticated: this is an AI Gateway key, which only works against the AI Gateway; create a platform API key in Settings → API Tokens and run `nbctl configure`")
	case !strings.HasPrefix(apiKey, apiKeyPrefix):
		return errors.New("not authenticated: this key predates the \"sk-nb-\" format, which current Nudgebee servers no longer accept; create a new key in Settings → API Tokens and run `nbctl configure`")
	default:
		return errors.New("not authenticated: the API key was rejected; it may have been deleted, suspended or expired")
	}
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
	runClient     *Client
	runClientOnce sync.Once
)

// getRunClient returns the singleton GraphQL client, initializing it if necessary.
func getRunClient() *Client {
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
func Run(ctx context.Context, req *Request, resp any) error {
	config.InitConfig()
	client := getRunClient()
	return client.Run(ctx, req, resp)
}

func (c *Client) Run(ctx context.Context, req *Request, resp any) error {
	// 1. Prepare payload
	payload := map[string]interface{}{
		"query":     req.q,
		"variables": req.vars,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal graphql payload: %w", err)
	}

	// 2. Create Request
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for k, v := range req.header {
		httpReq.Header[k] = v
	}

	// 3. Execute Request
	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer func() {
		_ = httpResp.Body.Close()
	}()

	if httpResp.StatusCode == http.StatusUnauthorized {
		return unauthorizedError(c.apiKey)
	}

	// 4. Decode Response
	// We want to handle errors specifically, so we decode into a raw map first or a struct with Errors.
	// We'll use a struct that captures Data as RawMessage to allow unmarshalling later.
	var graphQLResp struct {
		Data   json.RawMessage `json:"data"`
		Errors []interface{}   `json:"errors"`
	}

	// Stream directly to avoid double allocation (io.ReadAll + json.Unmarshal)
	if err := json.NewDecoder(httpResp.Body).Decode(&graphQLResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	// 6. Handle Errors
	if len(graphQLResp.Errors) > 0 {
		return c.handleGraphQLErrors(graphQLResp.Errors)
	}

	// 7. Unmarshal Data
	if resp == nil {
		return nil
	}

	if len(graphQLResp.Data) == 0 {
		// No data and no errors?
		return nil
	}

	// machinebox/graphql unmarshals into the struct.
	// If the user provided a struct, we need to be careful.
	// machinebox/graphql logic:
	// - if resp is a map, it just unmarshals data into it.
	// - if resp is a struct, it tries to match fields.

	// We will attempt to unmarshal data directly into resp.
	if err := json.Unmarshal(graphQLResp.Data, resp); err != nil {
		return fmt.Errorf("failed to unmarshal graphql data: %w", err)
	}

	return nil
}

func (c *Client) handleGraphQLErrors(errorsVal []interface{}) error {
	// Check for specific crash signature in error messages (Hasura ActionWebhookErrorResponse)
	for _, errItem := range errorsVal {
		if errMap, ok := errItem.(map[string]interface{}); ok {
			if msg, ok := errMap["message"].(string); ok {
				if strings.Contains(msg, "ActionWebhookErrorResponse") && strings.Contains(msg, "key \"message\" not found") {
					// Attempt to drill down into the actual error from the webhook
					// extensions.internal.response.body.errors[].message
					if extensions, ok := errMap["extensions"].(map[string]interface{}); ok {
						if internal, ok := extensions["internal"].(map[string]interface{}); ok {
							if response, ok := internal["response"].(map[string]interface{}); ok {
								if body, ok := response["body"].(map[string]interface{}); ok {
									if bodyErrs, ok := body["errors"].([]interface{}); ok && len(bodyErrs) > 0 {
										var sb strings.Builder
										sb.WriteString("Backend validation failed:\n")
										for _, be := range bodyErrs {
											if beMap, ok := be.(map[string]interface{}); ok {
												if beMsg, ok := beMap["message"].(string); ok {
													fmt.Fprintf(&sb, "- %s\n", beMsg)
												}
											}
										}
										return fmt.Errorf("%s", sb.String())
									}
								}
							}
						}
					}
					// Fallback if drill-down fails but it matched the signature
					if viper.GetBool("verbose") {
						jsonBytes, _ := json.MarshalIndent(errorsVal, "", "  ")
						return fmt.Errorf("validation failed (server error):\n%s", string(jsonBytes))
					}
					return fmt.Errorf("validation failed (server error). Run with --verbose to see full details")
				}
			}
		}
	}

	// Generic error handling
	// Convert errors to GraphQLErrors for backward compatibility or better typing?
	// The original code used a custom struct. Let's try to map it to GraphQLErrors if possible,
	// or just return a formatted error string.

	var errs GraphQLErrors
	jsonBytes, _ := json.Marshal(errorsVal)
	_ = json.Unmarshal(jsonBytes, &errs) // Best effort to unmarshal into our struct

	if len(errs) > 0 {
		return errs
	}

	// If unmarshalling failed (e.g. different structure), return JSON string
	formatted, _ := json.MarshalIndent(errorsVal, "", "  ")
	return fmt.Errorf("graphql errors:\n%s", string(formatted))
}
