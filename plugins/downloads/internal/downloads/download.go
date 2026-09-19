package downloads

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/woodleighschool/stemma-catalog/plugins/downloads/internal/discovery"
	"github.com/woodleighschool/stemma/plugin"
)

// Client bounds requests and refuses insecure redirects. Resolvers use public
// vendor endpoints; credentials and arbitrary request headers are not accepted.
func Client() *http.Client {
	return &http.Client{
		Timeout: 15 * time.Minute,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("too many redirects")
			}
			return validURL(req.URL.String())
		},
	}
}

func validURL(address string) error {
	u, err := url.Parse(address)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" || u.Port() != "" && u.Port() != "443" {
		return errors.New("download requires an HTTPS URL without credentials, fragments or custom ports")
	}
	return nil
}

func validateRelease(release discovery.Release) error {
	if err := validURL(release.URL); err != nil {
		return err
	}
	name := release.Filename
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\:\x00\r\n\t") || strings.TrimSpace(name) != name {
		return errors.New("download requires a safe filename")
	}
	if strings.ContainsAny(release.Version, "\x00\r\n\t") {
		return errors.New("invalid release version")
	}
	return nil
}

func download(ctx context.Context, client *http.Client, release discovery.Release, workspace string) (artifact plugin.Artifact, err error) {
	if err := validateRelease(release); err != nil {
		return artifact, err
	}
	if !filepath.IsAbs(workspace) {
		return artifact, errors.New("download requires an absolute leased workspace")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, release.URL, nil)
	if err != nil {
		return artifact, errors.New("create download request")
	}
	response, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return artifact, ctx.Err()
		}
		return artifact, errors.New("download request failed")
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return artifact, fmt.Errorf("download returned HTTP %d", response.StatusCode)
	}
	// Match the host's object bound without depending on its internal cache.
	const maxSize = 16 << 30
	if response.ContentLength > maxSize {
		return artifact, errors.New("download exceeds 16 GiB")
	}
	root, err := os.OpenRoot(workspace)
	if err != nil {
		return artifact, fmt.Errorf("open workspace: %w", err)
	}
	defer func() { _ = root.Close() }()
	file, err := root.OpenFile(release.Filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return artifact, fmt.Errorf("create artifact: %w", err)
	}
	defer func() {
		_ = file.Close()
		if err != nil {
			_ = root.Remove(release.Filename)
		}
	}()
	hash := sha256.New()
	size, err := io.Copy(io.MultiWriter(file, hash), io.LimitReader(response.Body, maxSize+1))
	if err != nil {
		return artifact, fmt.Errorf("read download: %w", err)
	}
	if size > maxSize {
		return artifact, errors.New("download exceeds 16 GiB")
	}
	if response.ContentLength >= 0 && size != response.ContentLength {
		return artifact, errors.New("download length differs from Content-Length")
	}
	if err := file.Close(); err != nil {
		return artifact, fmt.Errorf("close artifact: %w", err)
	}
	return plugin.Artifact{
		Path: filepath.Join(workspace, release.Filename), Filename: release.Filename,
		SHA256: hex.EncodeToString(hash.Sum(nil)), Size: size,
		Format:  strings.TrimPrefix(strings.ToLower(filepath.Ext(release.Filename)), "."),
		Version: release.Version,
	}, nil
}
