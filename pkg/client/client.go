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
	"strconv"
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

// sensitiveHeaders are never written to the verbose log. The log lands in the
// current directory, which may be kept or shared (e.g. a Nubi workspace).
var sensitiveHeaders = []string{"Authorization", "Proxy-Authorization", "Cookie", "Set-Cookie", "X-Api-Key"}

// redactHeaders returns a copy of h with sensitive values replaced.
func redactHeaders(h http.Header) http.Header {
	out := h.Clone()
	for _, name := range sensitiveHeaders {
		if out.Get(name) != "" {
			out.Set(name, "[REDACTED]")
		}
	}
	return out
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Log the request. Dump a clone with redacted headers, then send the clone
	// with the real headers: DumpRequestOut consumes and restores the body of
	// the request it is given, so the clone is the one with a readable body.
	logged := req.Clone(req.Context())
	logged.Header = redactHeaders(req.Header)
	reqDump, err := httputil.DumpRequestOut(logged, true)
	if err != nil {
		t.logger.Printf("Error dumping request: %v", err)
	} else {
		t.logger.Printf("Request:\n%s", reqDump)
	}
	logged.Header = req.Header.Clone()

	resp, err := t.wrapped.RoundTrip(logged)
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

	// Dump the response for logging, without sensitive headers.
	loggedResp := *resp
	loggedResp.Header = redactHeaders(resp.Header)
	respDump, dumpErr := httputil.DumpResponse(&loggedResp, true)
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
}

type ClientOption interface {
	apply(opts *clientOptions)
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

// resolveOptions applies opts and falls back to viper config and defaults.
// The returned endpoint has no trailing slash.
func resolveOptions(opts []ClientOption) clientOptions {
	config := clientOptions{}
	for _, o := range opts {
		o.apply(&config)
	}

	if config.endpoint == "" {
		config.endpoint = viper.GetString("endpoint")
	}
	if config.endpoint == "" {
		config.endpoint = "https://app.nudgebee.com"
	}
	config.endpoint = strings.TrimRight(config.endpoint, "/")

	if config.apiKey == "" {
		config.apiKey = viper.GetString("api-key")
	}
	return config
}

var (
	verboseLogger     *log.Logger
	verboseLoggerOnce sync.Once
)

// getVerboseLogger opens nbctl_graphql.log once per process, so long-running
// callers (e.g. the MCP server) that build many clients don't leak descriptors.
func getVerboseLogger() *log.Logger {
	verboseLoggerOnce.Do(func() {
		logFile, err := os.OpenFile("nbctl_graphql.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			log.Printf("Error opening log file: %v\n", err)
			return
		}
		verboseLogger = log.New(logFile, "", log.LstdFlags)
	})
	return verboseLogger
}

// newTransport returns the authenticating transport, wrapped in a request
// logger when --verbose is set.
func newTransport(apiKey string) http.RoundTripper {
	var transport http.RoundTripper = &authTransport{
		apiKey:  apiKey,
		wrapped: http.DefaultTransport,
	}

	if viper.GetBool("verbose") {
		if logger := getVerboseLogger(); logger != nil {
			transport = &loggingTransport{
				wrapped: transport,
				logger:  logger,
			}
		}
	}
	return transport
}

// DefaultHTTPTimeout bounds each HTTP request unless http-timeout is set.
const DefaultHTTPTimeout = 30 * time.Second

// httpTimeout reads http-timeout (flag --http-timeout, env NUDGEBEE_HTTP_TIMEOUT)
// as a Go duration ("50s", "2m") or a number of seconds. 0 disables the timeout.
// An invalid value falls back to the default with a warning.
func httpTimeout() time.Duration {
	raw := strings.TrimSpace(viper.GetString("http-timeout"))
	if raw == "" {
		return DefaultHTTPTimeout
	}
	if d, err := time.ParseDuration(raw); err == nil && d >= 0 {
		return d
	}
	if secs, err := strconv.ParseFloat(raw, 64); err == nil && secs >= 0 {
		return time.Duration(secs * float64(time.Second))
	}
	fmt.Fprintf(os.Stderr, "Warning: invalid http-timeout %q, using %s\n", raw, DefaultHTTPTimeout)
	return DefaultHTTPTimeout
}

func newHTTPClient(config clientOptions) *http.Client {
	return &http.Client{
		Transport: newTransport(config.apiKey),
		Timeout:   httpTimeout(),
	}
}

// NewClient creates a new GraphQL client.
func NewClient(opts ...ClientOption) *Client {
	config := resolveOptions(opts)
	return &Client{
		endpoint:   config.endpoint + "/api/graphql",
		apiKey:     config.apiKey,
		httpClient: newHTTPClient(config),
	}
}

// NewHTTPClient creates a new authenticated http.Client.
func NewHTTPClient(opts ...ClientOption) *http.Client {
	return newHTTPClient(resolveOptions(opts))
}

// ApiTokenPrefix is the prefix on every Nudgebee API token (`sk-nb-...`).
const ApiTokenPrefix = "sk-nb-"

// authTransport sends the configured API token as a Bearer on every request.
// The gateway authenticates the raw token directly; there is no exchange step.
type authTransport struct {
	apiKey  string
	wrapped http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.apiKey == "" {
		return nil, errors.New("no API key configured: run 'nbctl configure add' or set api-key")
	}
	// avoid mutating original request
	req2 := req.Clone(req.Context())
	req2.Header.Set("Authorization", "Bearer "+t.apiKey)
	return t.wrapped.RoundTrip(req2)
}

// serverErrorMessage extracts a human-readable message from an error body:
// GraphQL {"errors":[{"message"}]}, {"message"} or {"error"}. It returns ""
// when the body has none.
func serverErrorMessage(body []byte) string {
	var parsed struct {
		Message string `json:"message"`
		Error   any    `json:"error"`
		Errors  []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if json.Unmarshal(body, &parsed) != nil {
		return ""
	}
	var msgs []string
	for _, e := range parsed.Errors {
		if m := strings.TrimSpace(e.Message); m != "" {
			msgs = append(msgs, m)
		}
	}
	if len(msgs) > 0 {
		return strings.Join(msgs, "; ")
	}
	if m := strings.TrimSpace(parsed.Message); m != "" {
		return m
	}
	if m, ok := parsed.Error.(string); ok {
		return strings.TrimSpace(m)
	}
	return ""
}

// isBareUnauthorized reports a message that only restates the status or is a
// one-word code (the app gateway answers a bad API key with "Unauthorized" or
// "not_authenticated"); the API-key hint is more useful then.
func isBareUnauthorized(msg string) bool {
	m := strings.ToLower(strings.Trim(strings.TrimSpace(msg), ".!"))
	return m == "401 unauthorized" || !strings.ContainsAny(m, " \t")
}

// unauthorizedError explains a 401 from the gateway. The most likely causes are
// a deleted/expired token, or a token created before direct-token auth existed
// (those must be recreated).
func unauthorizedError(apiKey string) error {
	msg := "authentication failed (401 Unauthorized): the API key was rejected"
	if !strings.HasPrefix(apiKey, ApiTokenPrefix) {
		msg += fmt.Sprintf("; API keys must start with %q", ApiTokenPrefix)
	}
	return errors.New(msg + ". The token may be deleted, expired, or created before direct token auth was supported. " +
		"Create a new token under Settings → API Tokens and run 'nbctl configure add'")
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
		body, _ := io.ReadAll(io.LimitReader(httpResp.Body, 64<<10))
		if msg := serverErrorMessage(body); msg != "" && !isBareUnauthorized(msg) {
			// The server says why (e.g. an expired workspace token); the generic
			// API-key advice would point the wrong way.
			return fmt.Errorf("authentication failed (401 Unauthorized): %s", msg)
		}
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
