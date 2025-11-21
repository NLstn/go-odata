package framework

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// TestSuite represents a collection of compliance tests
type TestSuite struct {
	Name        string
	Description string
	SpecURL     string
	Tests       []Test
	Results     *TestResults
	ServerURL   string
	Debug       bool
	Verbose     bool
	Client      *http.Client
}

// Test represents a single test case
type Test struct {
	Name        string
	Description string
	Fn          func(*TestContext) error
}

// TestResults tracks the results of test execution
type TestResults struct {
	Total   int
	Passed  int
	Failed  int
	Skipped int
	Details []TestDetail
}

// TestDetail contains information about a single test result
type TestDetail struct {
	Name   string
	Status TestStatus
	Error  string
}

// TestStatus represents the status of a test
type TestStatus int

const (
	StatusPass TestStatus = iota
	StatusFail
	StatusSkip
)

func (s TestStatus) String() string {
	switch s {
	case StatusPass:
		return "PASS"
	case StatusFail:
		return "FAIL"
	case StatusSkip:
		return "SKIP"
	default:
		return "UNKNOWN"
	}
}

// TestContext provides context and utilities for a single test
type TestContext struct {
	suite  *TestSuite
	name   string
	buffer *bytes.Buffer
}

// HTTPResponse represents a complete HTTP response
type HTTPResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

// NewTestSuite creates a new test suite
func NewTestSuite(name, description, specURL string) *TestSuite {
	return &TestSuite{
		Name:        name,
		Description: description,
		SpecURL:     specURL,
		Tests:       []Test{},
		Results:     &TestResults{},
		ServerURL:   "http://localhost:9090",
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// AddTest adds a test to the suite
func (s *TestSuite) AddTest(name, description string, fn func(*TestContext) error) {
	s.Tests = append(s.Tests, Test{
		Name:        name,
		Description: description,
		Fn:          fn,
	})
}

// Run executes all tests in the suite
func (s *TestSuite) Run() error {
	if s.Verbose {
		fmt.Println("======================================")
		fmt.Printf("OData v4 Compliance Test\n")
		fmt.Printf("Suite: %s\n", s.Name)
		fmt.Println("======================================")
		fmt.Println()
		fmt.Printf("Description: %s\n", s.Description)
		fmt.Println()
		fmt.Printf("Spec Reference: %s\n", s.SpecURL)
		fmt.Println()
	} else {
		// In non-verbose mode, show a simple progress message
		fmt.Printf("Running %d tests... ", len(s.Tests))
	}

	for i, test := range s.Tests {
		s.Results.Total++
		ctx := &TestContext{
			suite:  s,
			name:   test.Name,
			buffer: &bytes.Buffer{},
		}

		// Reseed the database before each test to ensure clean state and isolation
		if err := s.reseedDatabase(); err != nil {
			if s.Verbose {
				fmt.Printf("\n⚠ WARNING: Failed to reseed database before test '%s': %v\n", test.Description, err)
				fmt.Println("Continuing with existing data...")
			}
		}

		err := test.Fn(ctx)
		if err != nil {
			if skipErr, ok := err.(*SkipError); ok {
				s.Results.Skipped++
				s.Results.Details = append(s.Results.Details, TestDetail{
					Name:   test.Description,
					Status: StatusSkip,
					Error:  skipErr.Reason,
				})
				if s.Verbose {
					fmt.Printf("\n⊘ SKIP: %s\n", test.Description)
					fmt.Printf("  Reason: %s\n", skipErr.Reason)
				}
			} else {
				s.Results.Failed++
				s.Results.Details = append(s.Results.Details, TestDetail{
					Name:   test.Description,
					Status: StatusFail,
					Error:  err.Error(),
				})
				if s.Verbose {
					fmt.Printf("\n✗ FAIL: %s\n", test.Description)
					fmt.Printf("  Details: %s\n", err.Error())
				}
			}
		} else {
			s.Results.Passed++
			s.Results.Details = append(s.Results.Details, TestDetail{
				Name:   test.Description,
				Status: StatusPass,
			})
			if s.Verbose {
				fmt.Printf("\n✓ PASS: %s\n", test.Description)
			}
		}

		// Print progress dots in non-verbose mode
		if !s.Verbose && (i+1)%10 == 0 {
			fmt.Printf("%d/%d ", i+1, len(s.Tests))
		}
	}

	if !s.Verbose {
		fmt.Printf("Done\n")
	}

	s.PrintSummary()

	if s.Results.Failed > 0 {
		return fmt.Errorf("%d test(s) failed", s.Results.Failed)
	}
	return nil
}

// PrintSummary prints the test summary in standardized format
func (s *TestSuite) PrintSummary() {
	fmt.Println()
	fmt.Println("======================================")
	fmt.Printf("COMPLIANCE_TEST_RESULT:PASSED=%d:FAILED=%d:SKIPPED=%d:TOTAL=%d\n",
		s.Results.Passed, s.Results.Failed, s.Results.Skipped, s.Results.Total)
	fmt.Println("======================================")

	if s.Results.Failed == 0 {
		fmt.Println("Status: PASSING")
	} else {
		fmt.Println("Status: FAILING")

		// Print failed tests list
		if !s.Verbose && s.Results.Failed > 0 {
			fmt.Println()
			fmt.Println("Failed Tests:")
			for _, detail := range s.Results.Details {
				if detail.Status == StatusFail {
					fmt.Printf("  ✗ %s\n", detail.Name)
					if detail.Error != "" {
						fmt.Printf("    Error: %s\n", detail.Error)
					}
				}
			}
		}
	}
}

// reseedDatabase calls the Reseed action to reset the database to a clean state
func (s *TestSuite) reseedDatabase() error {
	url := s.ServerURL + "/Reseed"
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := s.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call Reseed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Reseed returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// SkipError represents a skipped test
type SkipError struct {
	Reason string
}

func (e *SkipError) Error() string {
	return e.Reason
}

// Skip marks a test as skipped
func (c *TestContext) Skip(reason string) error {
	return &SkipError{Reason: reason}
}

// GET performs an HTTP GET request
func (c *TestContext) GET(path string, headers ...Header) (*HTTPResponse, error) {
	return c.request("GET", path, nil, headers...)
}

// POST performs an HTTP POST request
func (c *TestContext) POST(path string, body interface{}, headers ...Header) (*HTTPResponse, error) {
	return c.request("POST", path, body, headers...)
}

// PATCH performs an HTTP PATCH request
func (c *TestContext) PATCH(path string, body interface{}, headers ...Header) (*HTTPResponse, error) {
	return c.request("PATCH", path, body, headers...)
}

// PUT performs an HTTP PUT request
func (c *TestContext) PUT(path string, body interface{}, headers ...Header) (*HTTPResponse, error) {
	return c.request("PUT", path, body, headers...)
}

// DELETE performs an HTTP DELETE request
func (c *TestContext) DELETE(path string, headers ...Header) (*HTTPResponse, error) {
	return c.request("DELETE", path, nil, headers...)
}

// HEAD performs an HTTP HEAD request
func (c *TestContext) HEAD(path string, headers ...Header) (*HTTPResponse, error) {
	return c.request("HEAD", path, nil, headers...)
}

// POSTRaw performs an HTTP POST request with raw bytes and content type
func (c *TestContext) POSTRaw(path string, body []byte, contentType string, headers ...Header) (*HTTPResponse, error) {
	hdrs := append([]Header{{Key: "Content-Type", Value: contentType}}, headers...)
	return c.request("POST", path, body, hdrs...)
}

// PUTRaw performs an HTTP PUT request with raw bytes and content type
func (c *TestContext) PUTRaw(path string, body []byte, contentType string, headers ...Header) (*HTTPResponse, error) {
	hdrs := append([]Header{{Key: "Content-Type", Value: contentType}}, headers...)
	return c.request("PUT", path, body, hdrs...)
}

// GETWithHeaders performs an HTTP GET request with custom headers
func (c *TestContext) GETWithHeaders(path string, customHeaders map[string]string) (*HTTPResponse, error) {
	headers := make([]Header, 0, len(customHeaders))
	for k, v := range customHeaders {
		headers = append(headers, Header{Key: k, Value: v})
	}
	return c.request("GET", path, nil, headers...)
}

// Log logs a message during test execution
func (c *TestContext) Log(message string) {
	if c.suite.Debug {
		fmt.Printf("[LOG] %s\n", message)
	}
}

// ServerURL returns the base URL of the running compliance server
func (c *TestContext) ServerURL() string {
	return c.suite.ServerURL
}

// Header represents an HTTP header key-value pair
type Header struct {
	Key   string
	Value string
}

// request performs an HTTP request
func (c *TestContext) request(method, path string, body interface{}, headers ...Header) (*HTTPResponse, error) {
	// Normalize query strings so callers don't need to URL encode manually
	if strings.Contains(path, "?") {
		if parsed, err := url.Parse(path); err == nil {
			encodedPath := parsed.Path
			if rawQuery := parsed.Query().Encode(); rawQuery != "" {
				encodedPath += "?" + rawQuery
			}
			path = encodedPath
		}
	}

	url := c.suite.ServerURL + path

	var bodyReader io.Reader
	if body != nil {
		var bodyBytes []byte
		var err error
		switch v := body.(type) {
		case string:
			bodyBytes = []byte(v)
		case []byte:
			bodyBytes = v
		default:
			bodyBytes, err = json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal body: %w", err)
			}
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	contentTypeSet := false
	for _, h := range headers {
		if strings.EqualFold(h.Key, "Content-Type") {
			contentTypeSet = true
		}
		req.Header.Set(h.Key, h.Value)
	}
	if body != nil && !contentTypeSet {
		// Default to JSON for structured payloads
		req.Header.Set("Content-Type", "application/json")
	}

	if c.suite.Debug {
		c.debugRequest(req, body)
	}

	resp, err := c.suite.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform request: %w", err)
	}
	//nolint:errcheck
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	httpResp := &HTTPResponse{
		StatusCode: resp.StatusCode,
		Headers:    resp.Header,
		Body:       respBody,
	}

	if c.suite.Debug {
		c.debugResponse(httpResp)
	}

	return httpResp, nil
}

// debugRequest prints debug information about the request
func (c *TestContext) debugRequest(req *http.Request, body interface{}) {
	fmt.Println("\n╔══════════════════════════════════════════════════════╗")
	fmt.Println("║ DEBUG: HTTP Request                                  ║")
	fmt.Println("╚══════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Method: %s\n", req.Method)
	fmt.Printf("URL: %s\n", req.URL.String())
	if len(req.Header) > 0 {
		fmt.Println("Headers:")
		for k, v := range req.Header {
			fmt.Printf("  %s: %s\n", k, strings.Join(v, ", "))
		}
	}
	if body != nil {
		fmt.Println("Body:")
		if str, ok := body.(string); ok {
			fmt.Println(str)
		} else if b, ok := body.([]byte); ok {
			fmt.Println(string(b))
		} else {
			bodyJSON, err := json.MarshalIndent(body, "", "  ")
			if err == nil {
				fmt.Println(string(bodyJSON))
			}
		}
	}
	fmt.Println()
}

// debugResponse prints debug information about the response
func (c *TestContext) debugResponse(resp *HTTPResponse) {
	fmt.Println("\n╔══════════════════════════════════════════════════════╗")
	fmt.Println("║ DEBUG: HTTP Response                                 ║")
	fmt.Println("╚══════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Printf("Status Code: %d\n", resp.StatusCode)
	if len(resp.Body) > 0 {
		fmt.Println("Body:")
		// Try to pretty-print JSON
		var jsonBody interface{}
		if err := json.Unmarshal(resp.Body, &jsonBody); err == nil {
			prettyJSON, marshalErr := json.MarshalIndent(jsonBody, "", "  ")
			if marshalErr == nil {
				fmt.Println(string(prettyJSON))
			}
		} else {
			fmt.Println(string(resp.Body))
		}
	}
	fmt.Println()
}

// AssertStatusCode checks if the response has the expected status code
func (c *TestContext) AssertStatusCode(resp *HTTPResponse, expected int) error {
	if resp.StatusCode != expected {
		// Include response body in error message for debugging
		bodyPreview := string(resp.Body)
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200] + "..."
		}
		return fmt.Errorf("status code: %d (expected %d), body: %s", resp.StatusCode, expected, bodyPreview)
	}
	return nil
}

// AssertHeader checks if the response has the expected header value
func (c *TestContext) AssertHeader(resp *HTTPResponse, key, expected string) error {
	actual := resp.Headers.Get(key)
	if actual != expected {
		return fmt.Errorf("header %s: %q (expected %q)", key, actual, expected)
	}
	return nil
}

// AssertHeaderContains checks if the response header contains the expected substring
func (c *TestContext) AssertHeaderContains(resp *HTTPResponse, key, expected string) error {
	actual := resp.Headers.Get(key)
	if !strings.Contains(actual, expected) {
		return fmt.Errorf("header %s: %q does not contain %q", key, actual, expected)
	}
	return nil
}

// AssertJSONField checks if the response body contains the expected JSON field
func (c *TestContext) AssertJSONField(resp *HTTPResponse, field string) error {
	var data map[string]interface{}
	if err := json.Unmarshal(resp.Body, &data); err != nil {
		return fmt.Errorf("invalid JSON response: %w", err)
	}

	if _, ok := data[field]; !ok {
		return fmt.Errorf("field %q not found in JSON response", field)
	}
	return nil
}

// AssertBodyContains checks if the response body contains the expected string
func (c *TestContext) AssertBodyContains(resp *HTTPResponse, expected string) error {
	if !strings.Contains(string(resp.Body), expected) {
		return fmt.Errorf("expected %q not found in response body", expected)
	}
	return nil
}

// GetJSON unmarshals the response body into the provided interface
func (c *TestContext) GetJSON(resp *HTTPResponse, v interface{}) error {
	return json.Unmarshal(resp.Body, v)
}

// IsValidJSON checks if the response body is valid JSON
func (c *TestContext) IsValidJSON(resp *HTTPResponse) bool {
	var data interface{}
	return json.Unmarshal(resp.Body, &data) == nil
}

// NewError creates a new error with the given message
func NewError(message string) error {
	return fmt.Errorf("%s", message)
}

// ContainsAny checks if a string contains any of the provided substrings
func ContainsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}
