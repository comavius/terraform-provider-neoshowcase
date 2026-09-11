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
	sessionCookie := strings.TrimSpace(options.SessionCookie)
	if sessionCookie == "" {
		return nil, fmt.Errorf("session cookie must not be empty")
	}
	if strings.ContainsAny(sessionCookie, "\r\n") {
		return nil, fmt.Errorf("session cookie contains invalid characters")
	}

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
	httpClient.Transport = &sessionCookieRoundTripper{
		base:          httpClient.Transport,
		sessionCookie: sessionCookie,
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

func cloneHTTPClient(client *http.Client) *http.Client {
	clone := *client
	if clone.Transport == nil {
		clone.Transport = http.DefaultTransport
	}
	return &clone
}

type sessionCookieRoundTripper struct {
	base          http.RoundTripper
	sessionCookie string
}

func (r *sessionCookieRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	request = request.Clone(request.Context())
	request.Header = request.Header.Clone()
	request.Header.Set("Cookie", r.sessionCookie)
	return r.base.RoundTrip(request)
}
