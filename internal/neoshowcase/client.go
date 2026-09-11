package neoshowcase

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/traP-jp/terraform-provider-neoshowcase/internal/neoshowcase/gen"
	"github.com/traP-jp/terraform-provider-neoshowcase/internal/neoshowcase/gen/genconnect"
)

type Client struct {
	rpc genconnect.APIServiceClient
}

func NewClient(options Options) (*Client, error) {
	endpoint, err := normalizeEndpoint(options.Endpoint)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(options.User) == "" {
		return nil, fmt.Errorf("user must not be empty")
	}

	authHeader := strings.TrimSpace(options.AuthHeader)
	if authHeader == "" {
		authHeader = DefaultAuthHeader
	}
	if err := validateHeaderName(authHeader); err != nil {
		return nil, fmt.Errorf("invalid auth header: %w", err)
	}

	headers := make(http.Header, len(options.AdditionalHeaders)+1)
	for name, value := range options.AdditionalHeaders {
		if err := validateHeaderName(name); err != nil {
			return nil, fmt.Errorf("invalid additional header %q: %w", name, err)
		}
		headers.Set(name, value)
	}
	// The provider identity always wins over additional_headers.
	headers.Set(authHeader, options.User)

	httpClient := options.HTTPClient
	if httpClient == nil {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: options.InsecureSkipVerify, //nolint:gosec // Explicit provider option.
		}
		httpClient = &http.Client{Transport: transport}
	} else if options.InsecureSkipVerify {
		return nil, fmt.Errorf("insecure_skip_verify cannot be used with a custom HTTP client")
	}

	httpClient = cloneHTTPClient(httpClient)
	httpClient.Transport = &headerRoundTripper{
		base:    httpClient.Transport,
		headers: headers,
	}

	return &Client{
		rpc: genconnect.NewAPIServiceClient(httpClient, endpoint),
	}, nil
}

func (c *Client) GetMe(ctx context.Context) (*gen.User, error) {
	response, err := c.rpc.GetMe(ctx, connect.NewRequest(&emptypb.Empty{}))
	if err != nil {
		return nil, fmt.Errorf("get current NeoShowcase user: %w", err)
	}
	return response.Msg, nil
}

func (c *Client) GetSystemInfo(ctx context.Context) (*gen.SystemInfo, error) {
	response, err := c.rpc.GetSystemInfo(ctx, connect.NewRequest(&emptypb.Empty{}))
	if err != nil {
		return nil, fmt.Errorf("get NeoShowcase system info: %w", err)
	}
	return response.Msg, nil
}

func normalizeEndpoint(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("endpoint must not be empty")
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse endpoint: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("endpoint scheme must be http or https")
	}
	if parsed.Host == "" {
		return "", fmt.Errorf("endpoint host must not be empty")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("endpoint must not contain a query or fragment")
	}

	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return parsed.String(), nil
}

func validateHeaderName(name string) error {
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("name must not be empty")
	}
	if strings.ContainsAny(name, ":\r\n") {
		return fmt.Errorf("name contains invalid characters")
	}
	return nil
}

func cloneHTTPClient(client *http.Client) *http.Client {
	clone := *client
	if clone.Transport == nil {
		clone.Transport = http.DefaultTransport
	}
	return &clone
}

type headerRoundTripper struct {
	base    http.RoundTripper
	headers http.Header
}

func (r *headerRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	request = request.Clone(request.Context())
	request.Header = request.Header.Clone()
	for name, values := range r.headers {
		request.Header.Del(name)
		for _, value := range values {
			request.Header.Add(name, value)
		}
	}
	return r.base.RoundTrip(request)
}
