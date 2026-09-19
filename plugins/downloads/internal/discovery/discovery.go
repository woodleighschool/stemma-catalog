// Package discovery resolves vendor release metadata without downloading installers.
package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"unicode"
	"unicode/utf8"
)

const metadataLimit = 4 << 20

// Release identifies one installer selected from vendor metadata.
// Version is empty when the vendor does not supply an authoritative version.
type Release struct {
	URL      string `json:"url"`
	Filename string `json:"filename"`
	Version  string `json:"version"`
}

func metadata(ctx context.Context, client *http.Client, target *url.URL, allowed func(*url.URL) bool) ([]byte, error) {
	if !allowed(target) {
		return nil, errors.New("unexpected metadata URL")
	}
	if client == nil {
		return nil, errors.New("metadata HTTP client is required")
	}
	// Keep each vendor's redirect boundary local to this request without mutating
	// the client shared with artifact downloads or other discovery operations.
	bounded := *client
	bounded.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if !allowed(req.URL) {
			return errors.New("unexpected metadata redirect URL")
		}
		if len(via) >= 10 {
			return errors.New("too many metadata redirects")
		}
		if client.CheckRedirect != nil {
			return client.CheckRedirect(req, via)
		}
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create metadata request: %w", err)
	}
	req.Header.Set("Accept", "application/json, text/html;q=0.9, */*;q=0.1")
	req.Header.Set("User-Agent", "stemma-catalog-downloads")
	resp, err := bounded.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request metadata: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metadata returned HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, metadataLimit+1))
	if err != nil {
		return nil, fmt.Errorf("read metadata: %w", err)
	}
	if len(data) > metadataLimit {
		return nil, errors.New("metadata exceeds size limit")
	}
	return data, nil
}

func jsonMetadata(ctx context.Context, client *http.Client, target *url.URL, allowed func(*url.URL) bool, value any) error {
	data, err := metadata(ctx, client, target, allowed)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, value); err != nil {
		return errors.New("invalid metadata JSON")
	}
	return nil
}

func httpsURL(u *url.URL) bool {
	return u.Scheme == "https" && u.Host != "" && u.User == nil && u.Port() == "" &&
		u.Host == u.Hostname() && u.Fragment == "" && u.RawFragment == "" && u.Opaque == ""
}

func hostURL(host string) func(*url.URL) bool {
	return func(u *url.URL) bool {
		return httpsURL(u) && u.Host == host
	}
}

func validText(value string) bool {
	return value != "" && strings.TrimSpace(value) == value && utf8.ValidString(value) &&
		!strings.ContainsFunc(value, unicode.IsControl)
}

func validFilename(name string) bool {
	return validText(name) && len(name) <= 255 && name != "." && name != ".." &&
		!strings.HasSuffix(name, ".") && !strings.ContainsAny(name, `/\<>:"|?*`)
}
