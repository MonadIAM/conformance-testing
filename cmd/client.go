package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Client struct {
	base *url.URL
	http *http.Client
}

func newClient(baseURL, caFile, connectAddress string) (*Client, error) {
	base, err := url.Parse(baseURL)
	if err != nil ||
		base.Scheme != "https" ||
		base.Host == "" ||
		base.User != nil ||
		base.RawQuery != "" ||
		base.Fragment != "" ||
		(base.Path != "" && base.Path != "/") {
		return nil, fmt.Errorf("suite URL must be an HTTPS origin: %q", baseURL)
	}

	pem, err := os.ReadFile(caFile)
	if err != nil {
		return nil, err
	}

	roots, err := x509.SystemCertPool()
	if err != nil {
		roots = x509.NewCertPool()
	}

	if !roots.AppendCertsFromPEM(pem) {
		return nil, fmt.Errorf("no CA certificates in %s", caFile)
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.TLSClientConfig = &tls.Config{
		RootCAs:    roots,
		MinVersion: tls.VersionTLS12,
	}

	if connectAddress != "" {
		if _, _, err := net.SplitHostPort(connectAddress); err != nil {
			return nil, fmt.Errorf("connect address: %w", err)
		}

		dialer := &net.Dialer{Timeout: 10 * time.Second}

		transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, connectAddress)
		}
	}

	return &Client{
		base: base,
		http: &http.Client{
			Transport: transport,
			Timeout:   20 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}, nil
}

func (c *Client) Close() {
	c.http.CloseIdleConnections()
}

func (c *Client) Request(ctx context.Context, method, path string, query url.Values, body []byte) ([]byte, error) {
	u := *c.base
	u.Path = path
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, method, u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return data, err
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return data, fmt.Errorf("%s %s: HTTP %d (response saved in report)", method, path, res.StatusCode)
	}

	return data, nil
}
