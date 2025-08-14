package system

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// NetworkChecker provides network connectivity checking capabilities
type NetworkChecker struct {
	logger Logger
	client *http.Client
}

// NewNetworkChecker creates a new network checker
func NewNetworkChecker(logger Logger) *NetworkChecker {
	return &NetworkChecker{
		logger: logger,
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout: 5 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout: 5 * time.Second,
			},
		},
	}
}

// NetworkStatus represents network connectivity status
type NetworkStatus struct {
	Connected    bool              `json:"connected"`
	DNSWorking   bool              `json:"dns_working"`
	HTTPSWorking bool              `json:"https_working"`
	Latency      time.Duration     `json:"latency"`
	Error        string            `json:"error,omitempty"`
	Details      map[string]string `json:"details"`
}

// ConnectivityTest represents a connectivity test configuration
type ConnectivityTest struct {
	Name        string        `json:"name"`
	URL         string        `json:"url"`
	Timeout     time.Duration `json:"timeout"`
	ExpectedCode int          `json:"expected_code"`
	Required    bool          `json:"required"`
}

// NetworkDiagnostic contains comprehensive network diagnostic information
type NetworkDiagnostic struct {
	Timestamp     time.Time                    `json:"timestamp"`
	OverallStatus NetworkStatus                `json:"overall_status"`
	Tests         map[string]ConnectivityTest  `json:"tests"`
	Results       map[string]NetworkStatus     `json:"results"`
	Suggestions   []string                     `json:"suggestions"`
}

// CheckBasicConnectivity performs basic network connectivity checks
func (nc *NetworkChecker) CheckBasicConnectivity() *NetworkStatus {
	status := &NetworkStatus{
		Details: make(map[string]string),
	}
	
	start := time.Now()
	
	// Test DNS resolution
	if nc.logger != nil {
		nc.logger.Debug("Testing DNS resolution...")
	}
	
	_, err := net.LookupHost("google.com")
	if err != nil {
		status.Error = fmt.Sprintf("DNS resolution failed: %v", err)
		status.Details["dns_error"] = err.Error()
		return status
	}
	
	status.DNSWorking = true
	status.Details["dns_status"] = "working"
	
	// Test HTTP connectivity
	if nc.logger != nil {
		nc.logger.Debug("Testing HTTP connectivity...")
	}
	
	resp, err := nc.client.Get("https://www.google.com")
	if err != nil {
		status.Error = fmt.Sprintf("HTTP connectivity failed: %v", err)
		status.Details["http_error"] = err.Error()
		return status
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusOK {
		status.HTTPSWorking = true
		status.Connected = true
		status.Details["http_status"] = "working"
	} else {
		status.Error = fmt.Sprintf("HTTP request returned status: %d", resp.StatusCode)
		status.Details["http_status_code"] = fmt.Sprintf("%d", resp.StatusCode)
	}
	
	status.Latency = time.Since(start)
	status.Details["latency_ms"] = fmt.Sprintf("%.2f", float64(status.Latency.Nanoseconds())/1000000)
	
	return status
}

// CheckConnectivity tests connectivity to specific URLs
func (nc *NetworkChecker) CheckConnectivity(tests []ConnectivityTest) *NetworkDiagnostic {
	diagnostic := &NetworkDiagnostic{
		Timestamp: time.Now(),
		Tests:     make(map[string]ConnectivityTest),
		Results:   make(map[string]NetworkStatus),
		Suggestions: []string{},
	}
	
	// Store test configurations
	for _, test := range tests {
		diagnostic.Tests[test.Name] = test
	}
	
	// Perform basic connectivity check first
	diagnostic.OverallStatus = *nc.CheckBasicConnectivity()
	
	if !diagnostic.OverallStatus.Connected {
		diagnostic.Suggestions = append(diagnostic.Suggestions, 
			"Check your internet connection",
			"Verify network settings",
			"Check firewall configuration")
		return diagnostic
	}
	
	// Run individual tests
	var failedRequired []string
	var failedOptional []string
	
	for _, test := range tests {
		if nc.logger != nil {
			nc.logger.Debug("Testing connectivity to: %s", test.URL)
		}
		
		result := nc.testURL(test)
		diagnostic.Results[test.Name] = result
		
		if !result.Connected {
			if test.Required {
				failedRequired = append(failedRequired, test.Name)
			} else {
				failedOptional = append(failedOptional, test.Name)
			}
		}
	}
	
	// Generate suggestions based on failures
	if len(failedRequired) > 0 {
		diagnostic.Suggestions = append(diagnostic.Suggestions,
			fmt.Sprintf("Critical services unreachable: %s", strings.Join(failedRequired, ", ")),
			"Check if these services are blocked by firewall or proxy")
	}
	
	if len(failedOptional) > 0 {
		diagnostic.Suggestions = append(diagnostic.Suggestions,
			fmt.Sprintf("Optional services unreachable: %s", strings.Join(failedOptional, ", ")))
	}
	
	return diagnostic
}

// testURL tests connectivity to a specific URL
func (nc *NetworkChecker) testURL(test ConnectivityTest) NetworkStatus {
	status := NetworkStatus{
		Details: make(map[string]string),
	}
	
	start := time.Now()
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), test.Timeout)
	defer cancel()
	
	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", test.URL, nil)
	if err != nil {
		status.Error = fmt.Sprintf("Failed to create request: %v", err)
		return status
	}
	
	// Set user agent
	req.Header.Set("User-Agent", "ApkHub-CLI/1.0")
	
	// Perform request
	resp, err := nc.client.Do(req)
	if err != nil {
		status.Error = fmt.Sprintf("Request failed: %v", err)
		status.Details["error_type"] = nc.categorizeNetworkError(err)
		return status
	}
	defer resp.Body.Close()
	
	status.Latency = time.Since(start)
	status.Details["latency_ms"] = fmt.Sprintf("%.2f", float64(status.Latency.Nanoseconds())/1000000)
	status.Details["status_code"] = fmt.Sprintf("%d", resp.StatusCode)
	
	// Check if status code matches expectation
	expectedCode := test.ExpectedCode
	if expectedCode == 0 {
		expectedCode = http.StatusOK
	}
	
	if resp.StatusCode == expectedCode {
		status.Connected = true
		status.HTTPSWorking = true
	} else {
		status.Error = fmt.Sprintf("Unexpected status code: %d (expected %d)", resp.StatusCode, expectedCode)
	}
	
	return status
}

// categorizeNetworkError categorizes network errors for better diagnostics
func (nc *NetworkChecker) categorizeNetworkError(err error) string {
	errStr := strings.ToLower(err.Error())
	
	switch {
	case strings.Contains(errStr, "timeout"):
		return "timeout"
	case strings.Contains(errStr, "connection refused"):
		return "connection_refused"
	case strings.Contains(errStr, "no such host"):
		return "dns_failure"
	case strings.Contains(errStr, "network unreachable"):
		return "network_unreachable"
	case strings.Contains(errStr, "certificate"):
		return "tls_certificate_error"
	case strings.Contains(errStr, "proxy"):
		return "proxy_error"
	default:
		return "unknown"
	}
}

// GetDefaultConnectivityTests returns default connectivity tests for ApkHub
func GetDefaultConnectivityTests() []ConnectivityTest {
	return []ConnectivityTest{
		{
			Name:         "Google DNS",
			URL:          "https://dns.google",
			Timeout:      5 * time.Second,
			ExpectedCode: http.StatusOK,
			Required:     true,
		},
		{
			Name:         "GitHub",
			URL:          "https://github.com",
			Timeout:      10 * time.Second,
			ExpectedCode: http.StatusOK,
			Required:     false,
		},
		{
			Name:         "Maven Central",
			URL:          "https://repo1.maven.org/maven2/",
			Timeout:      10 * time.Second,
			ExpectedCode: http.StatusOK,
			Required:     false,
		},
	}
}

// DiagnoseNetworkIssue provides detailed diagnosis for network issues
func (nc *NetworkChecker) DiagnoseNetworkIssue(err error) []string {
	var suggestions []string
	
	if err == nil {
		return suggestions
	}
	
	errStr := strings.ToLower(err.Error())
	
	switch {
	case strings.Contains(errStr, "timeout"):
		suggestions = append(suggestions,
			"Network request timed out",
			"Check your internet connection speed",
			"Try increasing timeout values",
			"Check if the server is responding slowly")
			
	case strings.Contains(errStr, "connection refused"):
		suggestions = append(suggestions,
			"Connection was refused by the server",
			"Check if the service is running",
			"Verify the URL and port are correct",
			"Check firewall settings")
			
	case strings.Contains(errStr, "no such host"):
		suggestions = append(suggestions,
			"DNS resolution failed",
			"Check your DNS settings",
			"Try using a different DNS server (8.8.8.8, 1.1.1.1)",
			"Verify the hostname is correct")
			
	case strings.Contains(errStr, "network unreachable"):
		suggestions = append(suggestions,
			"Network is unreachable",
			"Check your network connection",
			"Verify network interface is up",
			"Check routing configuration")
			
	case strings.Contains(errStr, "certificate"):
		suggestions = append(suggestions,
			"TLS certificate error",
			"Check system date and time",
			"Update CA certificates",
			"Verify the server's SSL certificate")
			
	case strings.Contains(errStr, "proxy"):
		suggestions = append(suggestions,
			"Proxy configuration issue",
			"Check proxy settings",
			"Verify proxy authentication",
			"Try bypassing proxy for this request")
			
	default:
		suggestions = append(suggestions,
			"Unknown network error occurred",
			"Check your internet connection",
			"Verify firewall and proxy settings",
			"Try the operation again later")
	}
	
	return suggestions
}

// TestProxyConfiguration tests if proxy configuration is working
func (nc *NetworkChecker) TestProxyConfiguration(proxyURL string) *NetworkStatus {
	status := &NetworkStatus{
		Details: make(map[string]string),
	}
	
	if proxyURL == "" {
		status.Error = "No proxy URL provided"
		return status
	}
	
	// Parse proxy URL
	parsedURL, err := url.Parse(proxyURL)
	if err != nil {
		status.Error = fmt.Sprintf("Invalid proxy URL: %v", err)
		return status
	}
	
	status.Details["proxy_url"] = proxyURL
	status.Details["proxy_scheme"] = parsedURL.Scheme
	status.Details["proxy_host"] = parsedURL.Host
	
	// Create client with proxy
	transport := &http.Transport{
		Proxy: http.ProxyURL(parsedURL),
		DialContext: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout: 5 * time.Second,
	}
	
	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}
	
	// Test connectivity through proxy
	start := time.Now()
	resp, err := client.Get("https://httpbin.org/ip")
	if err != nil {
		status.Error = fmt.Sprintf("Proxy test failed: %v", err)
		status.Details["error_type"] = nc.categorizeNetworkError(err)
		return status
	}
	defer resp.Body.Close()
	
	status.Latency = time.Since(start)
	status.Connected = resp.StatusCode == http.StatusOK
	status.HTTPSWorking = status.Connected
	status.Details["status_code"] = fmt.Sprintf("%d", resp.StatusCode)
	status.Details["latency_ms"] = fmt.Sprintf("%.2f", float64(status.Latency.Nanoseconds())/1000000)
	
	return status
}

// FormatNetworkDiagnostic formats network diagnostic information for display
func (nc *NetworkChecker) FormatNetworkDiagnostic(diagnostic *NetworkDiagnostic) string {
	if diagnostic == nil {
		return "No network diagnostic information available"
	}
	
	output := fmt.Sprintf("Network Diagnostic Report (as of %s):\n", 
		diagnostic.Timestamp.Format("2006-01-02 15:04:05"))
	
	// Overall status
	output += fmt.Sprintf("\nOverall Status:\n")
	if diagnostic.OverallStatus.Connected {
		output += fmt.Sprintf("  ✅ Connected (latency: %v)\n", diagnostic.OverallStatus.Latency)
	} else {
		output += fmt.Sprintf("  ❌ Not connected: %s\n", diagnostic.OverallStatus.Error)
	}
	
	output += fmt.Sprintf("  DNS: %v\n", diagnostic.OverallStatus.DNSWorking)
	output += fmt.Sprintf("  HTTPS: %v\n", diagnostic.OverallStatus.HTTPSWorking)
	
	// Individual test results
	if len(diagnostic.Results) > 0 {
		output += fmt.Sprintf("\nConnectivity Tests:\n")
		for name, result := range diagnostic.Results {
			test := diagnostic.Tests[name]
			status := "❌"
			if result.Connected {
				status = "✅"
			}
			
			required := ""
			if test.Required {
				required = " (required)"
			}
			
			output += fmt.Sprintf("  %s %s%s: ", status, name, required)
			if result.Connected {
				output += fmt.Sprintf("OK (%.2fms)\n", float64(result.Latency.Nanoseconds())/1000000)
			} else {
				output += fmt.Sprintf("Failed - %s\n", result.Error)
			}
		}
	}
	
	// Suggestions
	if len(diagnostic.Suggestions) > 0 {
		output += fmt.Sprintf("\nSuggestions:\n")
		for i, suggestion := range diagnostic.Suggestions {
			output += fmt.Sprintf("  %d. %s\n", i+1, suggestion)
		}
	}
	
	return output
}