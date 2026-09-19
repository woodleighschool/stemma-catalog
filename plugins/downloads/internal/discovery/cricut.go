package discovery

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// CricutConfig selects the macOS rollout shard for Cricut Design Space.
type CricutConfig struct {
	Shard string `json:"shard,omitempty" jsonschema:"pattern=^[A-Za-z0-9_-]+$,default=a" jsonschema_description:"Vendor rollout shard used to select the update manifest. Defaults to the primary shard a."`
}

// Cricut resolves the vendor's update JSON and installer indirection.
// The protocol does not provide an authoritative installer version.
func Cricut(ctx context.Context, client *http.Client, config CricutConfig) (Release, error) {
	query := url.Values{"operatingSystem": {"osxnative"}, "shard": {config.Shard}}
	endpoint := &url.URL{Scheme: "https", Host: "apis.cricut.com", Path: "/desktopdownload/UpdateJson", RawQuery: query.Encode()}
	var update struct {
		Result string `json:"result"`
	}
	if err := jsonMetadata(ctx, client, endpoint, cricutURL, &update); err != nil {
		return Release{}, fmt.Errorf("cricut: update location: %w", err)
	}
	manifest, err := url.Parse(update.Result)
	if err != nil || !cricutURL(manifest) {
		return Release{}, errors.New("cricut: unexpected update manifest URL")
	}
	var rollout struct {
		InstallFile string `json:"rolloutInstallFile"`
	}
	if err := jsonMetadata(ctx, client, manifest, cricutURL, &rollout); err != nil {
		return Release{}, fmt.Errorf("cricut: rollout: %w", err)
	}
	if !validFilename(rollout.InstallFile) || !strings.HasSuffix(rollout.InstallFile, ".dmg") {
		return Release{}, errors.New("cricut: rollout installer must be a DMG filename")
	}
	query.Set("fileName", rollout.InstallFile)
	endpoint.Path = "/desktopdownload/InstallerFile"
	endpoint.RawQuery = query.Encode()
	var installer struct {
		Result string `json:"result"`
	}
	if err := jsonMetadata(ctx, client, endpoint, cricutURL, &installer); err != nil {
		return Release{}, fmt.Errorf("cricut: installer location: %w", err)
	}
	download, err := url.Parse(installer.Result)
	if err != nil || !cricutURL(download) {
		return Release{}, errors.New("cricut: unexpected installer URL")
	}
	filename := path.Base(download.Path)
	if !validFilename(filename) || filename != rollout.InstallFile || strings.HasSuffix(download.Path, "/") || path.Clean(download.Path) != download.Path {
		return Release{}, errors.New("cricut: installer URL does not match rollout filename")
	}
	return Release{URL: download.String(), Filename: filename}, nil
}

func cricutURL(u *url.URL) bool {
	return httpsURL(u) && (u.Host == "cricut.com" || strings.HasSuffix(u.Host, ".cricut.com"))
}
