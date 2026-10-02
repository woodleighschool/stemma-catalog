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
	address, err := cricutInstallerURL(ctx, client, config, rollout.InstallFile)
	if err != nil {
		return Release{}, err
	}
	return Release{URL: address, Filename: rollout.InstallFile, Signed: true}, nil
}

// CricutDownload refreshes the signed URL for the recorded installer without
// selecting the current rollout, which may have advanced since discovery.
func CricutDownload(ctx context.Context, client *http.Client, config CricutConfig, release Release) (string, error) {
	if release.URL != "" || release.Version != "" || !validFilename(release.Filename) || !strings.HasSuffix(release.Filename, ".dmg") {
		return "", errors.New("cricut: invalid installer observation")
	}
	return cricutInstallerURL(ctx, client, config, release.Filename)
}

func cricutInstallerURL(ctx context.Context, client *http.Client, config CricutConfig, filename string) (string, error) {
	query := url.Values{"operatingSystem": {"osxnative"}, "shard": {config.Shard}, "fileName": {filename}}
	endpoint := &url.URL{Scheme: "https", Host: "apis.cricut.com", Path: "/desktopdownload/InstallerFile", RawQuery: query.Encode()}
	var installer struct {
		Result string `json:"result"`
	}
	if err := jsonMetadata(ctx, client, endpoint, cricutURL, &installer); err != nil {
		return "", fmt.Errorf("cricut: installer location: %w", err)
	}
	download, err := url.Parse(installer.Result)
	if err != nil || !cricutURL(download) {
		return "", errors.New("cricut: unexpected installer URL")
	}
	if path.Base(download.Path) != filename || strings.HasSuffix(download.Path, "/") || path.Clean(download.Path) != download.Path {
		return "", errors.New("cricut: installer URL does not match recorded filename")
	}
	return download.String(), nil
}

func cricutURL(u *url.URL) bool {
	return httpsURL(u) && (u.Host == "cricut.com" || strings.HasSuffix(u.Host, ".cricut.com"))
}
