// Feed IDs and Office updater selection adapted from AutoPkg's
// MSOfficeMacURLandUpdateInfoProvider.py, Copyright 2015 Allister Banks and
// Tim Sutton, licensed under the Apache License, Version 2.0.
// https://github.com/autopkg/recipes/blob/master/MSOfficeUpdates/MSOfficeMacURLandUpdateInfoProvider.py

package discovery

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"howett.net/plist"
)

// MicrosoftConfig selects a macOS product, release channel and package type.
// Channel defaults to production and Type defaults to standalone.
type MicrosoftConfig struct {
	Product string `json:"product"`
	Channel string `json:"channel"`
	Type    string `json:"type"`
}

var microsoftApps = map[string]string{
	"excel": "XCEL2019", "onenote": "ONMC2019", "outlook": "OPIM2019",
	"powerpoint": "PPT32019", "word": "MSWD2019",
}

var microsoftChannels = map[string]string{
	"production": "C1297A47-86C4-4C1F-97FA-950631F94777",
	"preview":    "1ac37578-5a24-40fb-892e-b89d85b6dfaa",
	"beta":       "4B2D7701-0A4F-49C8-B4CB-0C2D4043F51F",
}

// These products publish standalone installers separately from MAU updaters.
// Edge no longer publishes current releases through its MAU feed.
var microsoftInstallers = map[string]string{
	"office": "2009112", "defender": "2097502", "edge": "2093504",
	"teams": "2249065", "company-portal": "853070", "onedrive": "823060",
	"windows-app": "868963",
}

// Microsoft resolves Office's MAU feed or a product's standalone download link.
func Microsoft(ctx context.Context, client *http.Client, config MicrosoftConfig) (Release, error) {
	if config.Channel == "" {
		config.Channel = "production"
	}
	if config.Type == "" {
		config.Type = "standalone"
	}
	channel, ok := microsoftChannels[config.Channel]
	if !ok {
		return Release{}, errors.New("microsoft: channel must be production, preview or beta")
	}
	if config.Type != "standalone" && config.Type != "updater" {
		return Release{}, errors.New("microsoft: type must be standalone or updater")
	}
	if link, ok := microsoftInstallers[config.Product]; ok {
		if config.Channel != "production" || config.Type != "standalone" {
			return Release{}, fmt.Errorf("microsoft: %s supports production standalone installers", config.Product)
		}
		return microsoftInstaller(ctx, client, link)
	}
	app, ok := microsoftApps[config.Product]
	if !ok {
		return Release{}, fmt.Errorf("microsoft: unknown product %q", config.Product)
	}
	endpoint := &url.URL{Scheme: "https", Host: "res.public.onecdn.static.microsoft", Path: "/mro1cdnstorage/" + channel + "/MacAutoupdate/0409" + app + ".xml"}
	data, err := metadata(ctx, client, endpoint, microsoftURL)
	if err != nil {
		return Release{}, fmt.Errorf("microsoft: manifest: %w", err)
	}
	var updates []struct {
		Location            string `plist:"Location"`
		FullUpdaterLocation string `plist:"FullUpdaterLocation"`
		Version             string `plist:"Update Version"`
	}
	if _, err := plist.Unmarshal(data, &updates); err != nil {
		return Release{}, fmt.Errorf("microsoft: manifest plist: %w", err)
	}
	// MAU orders current releases before releases retained for older macOS
	// versions. Delta entries require an existing version and are not selected.
	for _, update := range updates {
		if strings.TrimSpace(update.FullUpdaterLocation) != "" {
			continue
		}
		u, err := url.Parse(strings.TrimSpace(update.Location))
		if err != nil || !microsoftURL(u) {
			return Release{}, errors.New("microsoft: unexpected package URL")
		}
		// OneNote's official standalone link points to its full updater package.
		if config.Type == "standalone" && config.Product != "onenote" {
			base, ok := strings.CutSuffix(u.Path, "_Updater.pkg")
			if !ok {
				return Release{}, errors.New("microsoft: Office updater URL must end in _Updater.pkg")
			}
			u.Path, u.RawPath = base+"_Installer.pkg", ""
		}
		version := strings.TrimSpace(update.Version)
		if !validText(version) {
			return Release{}, errors.New("microsoft: manifest has no valid update version")
		}
		return microsoftPackage(u, version)
	}
	return Release{}, errors.New("microsoft: manifest has no full update")
}

func microsoftInstaller(ctx context.Context, client *http.Client, link string) (Release, error) {
	if client == nil {
		return Release{}, errors.New("microsoft: HTTP client is required")
	}
	bounded := *client
	bounded.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if !microsoftURL(req.URL) || len(via) >= 10 {
			return errors.New("microsoft: unexpected installer redirect")
		}
		if client.CheckRedirect != nil {
			return client.CheckRedirect(req, via)
		}
		return nil
	}
	// Resolve the public download link without reading installer bytes.
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, "https://go.microsoft.com/fwlink/?linkid="+link, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("User-Agent", "stemma-catalog-downloads")
	response, err := bounded.Do(req)
	if err != nil {
		return Release{}, fmt.Errorf("microsoft: standalone location: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("microsoft: standalone location returned HTTP %d", response.StatusCode)
	}
	return microsoftPackage(response.Request.URL, "")
}

func microsoftPackage(u *url.URL, version string) (Release, error) {
	filename := path.Base(u.Path)
	if !microsoftURL(u) || !validFilename(filename) || !strings.HasSuffix(filename, ".pkg") {
		return Release{}, errors.New("microsoft: expected a Microsoft PKG URL")
	}
	return Release{URL: u.String(), Filename: filename, Version: version}, nil
}

func microsoftURL(u *url.URL) bool {
	return httpsURL(u) && (strings.HasSuffix(u.Host, ".microsoft.com") ||
		strings.HasSuffix(u.Host, ".static.microsoft") || u.Host == "oneclient.sfx.ms" ||
		u.Host == "statics.teams.cdn.office.net")
}
