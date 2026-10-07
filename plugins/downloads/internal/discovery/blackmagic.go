// Blackmagic's download protocol follows AutoPkg's BlackMagicURLProvider.py,
// Copyright 2014 Timothy Sutton, licensed under the Apache License, Version 2.0.
// https://github.com/autopkg/timsutton-recipes/blob/master/Blackmagic/BlackMagicURLProvider.py

package discovery

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
)

// BlackmagicConfig selects a vendor product, platform and release-name pattern.
type BlackmagicConfig struct {
	Product      string                  `json:"product" jsonschema:"minLength=1" jsonschema_description:"Product name sent to Blackmagic, such as DaVinci Resolve or DaVinci Resolve Studio."`
	Platform     string                  `json:"platform,omitempty" jsonschema:"default=Mac OS X" jsonschema_description:"Platform key in the vendor feed, such as Mac OS X, Windows or Linux."`
	NamePattern  string                  `json:"name_pattern" jsonschema:"minLength=1" jsonschema_description:"Regular expression matching release names, with a named version group containing a dotted numeric version. Determines edition, release channel and any major-version policy."`
	Registration *BlackmagicRegistration `json:"registration,omitempty" jsonschema_description:"Registration details sent to Blackmagic when acquiring the recorded installer."`
}

type BlackmagicRegistration struct {
	Firstname string `json:"firstname" jsonschema:"minLength=1"`
	Lastname  string `json:"lastname" jsonschema:"minLength=1"`
	Email     string `json:"email" jsonschema:"minLength=1"`
	Phone     string `json:"phone" jsonschema:"minLength=1"`
	Street    string `json:"street" jsonschema:"minLength=1"`
	City      string `json:"city" jsonschema:"minLength=1"`
	Country   string `json:"country" jsonschema:"pattern=^[a-z]{2}$" jsonschema_description:"Two-letter lowercase country code, such as au."`
	Policy    bool   `json:"policy,omitempty" jsonschema:"default=false" jsonschema_description:"Consent to occasional Blackmagic software update, product and service emails. Defaults to false."`
}

func (c BlackmagicConfig) Validate() error {
	if !validText(c.Product) || !validText(c.Platform) {
		return errors.New("blackmagic: product and platform require nonblank text")
	}
	pattern, err := regexp.Compile(c.NamePattern)
	if err != nil || pattern.SubexpIndex("version") < 0 {
		return errors.New("blackmagic: name_pattern requires a valid regular expression with a named version group")
	}
	if c.Registration == nil {
		return nil
	}
	for _, field := range []struct{ name, value string }{
		{"firstname", c.Registration.Firstname}, {"lastname", c.Registration.Lastname},
		{"email", c.Registration.Email}, {"phone", c.Registration.Phone},
		{"street", c.Registration.Street}, {"city", c.Registration.City}, {"country", c.Registration.Country},
	} {
		if !validText(field.value) {
			return fmt.Errorf("blackmagic: registration.%s requires nonblank text without surrounding whitespace or control characters", field.name)
		}
	}
	return nil
}

// BlackmagicRelease records vendor identity, not an expiring download URL or registration data.
type BlackmagicRelease struct {
	DownloadID           string `json:"download_id"`
	Version              string `json:"version"`
	RequiresRegistration bool   `json:"requires_registration"`
}

var blackmagicVersion = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)*$`)
var blackmagicID = regexp.MustCompile(`^[a-f0-9]{32}$`)

// Blackmagic selects the highest matching version with a download for the requested platform.
func Blackmagic(ctx context.Context, client *http.Client, config BlackmagicConfig) (BlackmagicRelease, error) {
	if err := config.Validate(); err != nil {
		return BlackmagicRelease{}, err
	}
	pattern, _ := regexp.Compile(config.NamePattern)
	target := &url.URL{Scheme: "https", Host: "www.blackmagicdesign.com", Path: "/api/support/us/downloads.json"}
	type download struct {
		ID string `json:"downloadId"`
	}
	var feed struct {
		Downloads []struct {
			Name                 string                `json:"name"`
			URLs                 map[string][]download `json:"urls"`
			RequiresRegistration bool                  `json:"requiresRegistration"`
		} `json:"downloads"`
	}
	if err := jsonMetadata(ctx, client, target, hostURL(target.Host), &feed); err != nil {
		return BlackmagicRelease{}, fmt.Errorf("blackmagic: releases: %w", err)
	}
	var selected BlackmagicRelease
	var candidates []download
	ambiguous := false
	for _, entry := range feed.Downloads {
		match := pattern.FindStringSubmatch(entry.Name)
		if match == nil || len(entry.URLs[config.Platform]) == 0 {
			continue
		}
		version := match[pattern.SubexpIndex("version")]
		if !blackmagicVersion.MatchString(version) || len(version) > 64 {
			return BlackmagicRelease{}, errors.New("blackmagic: matched release requires a dotted numeric version")
		}
		comparison := compareBlackmagicVersions(version, selected.Version)
		if selected.Version != "" && comparison < 0 {
			continue
		}
		current := entry.URLs[config.Platform]
		if selected.Version != "" && comparison == 0 {
			if len(candidates) != 1 || len(current) != 1 || current[0].ID != candidates[0].ID {
				ambiguous = true
			}
			continue
		}
		selected = BlackmagicRelease{Version: version, RequiresRegistration: entry.RequiresRegistration}
		candidates = current
		ambiguous = false
	}
	if selected.Version == "" {
		return BlackmagicRelease{}, errors.New("blackmagic: no release matches name_pattern and platform")
	}
	if ambiguous || len(candidates) != 1 || !blackmagicID.MatchString(candidates[0].ID) {
		return BlackmagicRelease{}, errors.New("blackmagic: selected release requires one unambiguous download ID")
	}
	selected.DownloadID = candidates[0].ID
	return selected, nil
}

func compareBlackmagicVersions(a, b string) int {
	left, right := strings.Split(a, "."), strings.Split(b, ".")
	for i := 0; i < max(len(left), len(right)); i++ {
		x, y := "", ""
		if i < len(left) {
			x = strings.TrimLeft(left[i], "0")
		}
		if i < len(right) {
			y = strings.TrimLeft(right[i], "0")
		}
		if len(x) < len(y) {
			return -1
		}
		if len(x) > len(y) {
			return 1
		}
		if n := strings.Compare(x, y); n != 0 {
			return n
		}
	}
	return 0
}

// BlackmagicDownload registers for the recorded release without consulting the current feed.
func BlackmagicDownload(ctx context.Context, client *http.Client, config BlackmagicConfig, release BlackmagicRelease) (Release, error) {
	if !blackmagicID.MatchString(release.DownloadID) || !blackmagicVersion.MatchString(release.Version) || len(release.Version) > 64 {
		return Release{}, errors.New("blackmagic: invalid installer observation")
	}
	if err := config.Validate(); err != nil {
		return Release{}, err
	}
	if release.RequiresRegistration && config.Registration == nil {
		return Release{}, errors.New("blackmagic: selected download requires registration")
	}
	country := "us"
	policy := false
	if config.Registration != nil {
		country = config.Registration.Country
		policy = config.Registration.Policy
	}
	payload := struct {
		*BlackmagicRegistration
		Platform string `json:"platform"`
		Product  string `json:"product"`
		Country  string `json:"country"`
		Policy   bool   `json:"policy"`
	}{config.Registration, config.Platform, config.Product, country, policy}
	data, err := json.Marshal(payload)
	if err != nil {
		return Release{}, fmt.Errorf("blackmagic: encode registration: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://www.blackmagicdesign.com/api/register/us/download/"+release.DownloadID, bytes.NewReader(data))
	if err != nil {
		return Release{}, fmt.Errorf("blackmagic: create registration request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	if client == nil {
		return Release{}, errors.New("blackmagic: HTTP client is required")
	}
	bounded := *client
	// Registration details must not be forwarded through redirects.
	bounded.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	resp, err := bounded.Do(req)
	if err != nil {
		return Release{}, fmt.Errorf("blackmagic: request download: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("blackmagic: registration returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8193))
	if err != nil {
		return Release{}, fmt.Errorf("blackmagic: read download URL: %w", err)
	}
	if len(body) > 8192 {
		return Release{}, errors.New("blackmagic: download URL exceeds size limit")
	}
	address, err := url.Parse(strings.TrimSpace(string(body)))
	if err != nil || !httpsURL(address) || address.Host != "blackmagicdesign.com" && !strings.HasSuffix(address.Host, ".blackmagicdesign.com") || !validFilename(path.Base(address.Path)) || strings.HasSuffix(address.Path, "/") {
		return Release{}, errors.New("blackmagic: expected a Blackmagic HTTPS installer URL")
	}
	return Release{URL: address.String(), Filename: path.Base(address.Path), Version: release.Version}, nil
}
