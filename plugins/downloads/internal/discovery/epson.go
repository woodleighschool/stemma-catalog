package discovery

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// EpsonConfig selects one Download Center content type for a device and OS.
type EpsonConfig struct {
	DeviceID string `json:"device_id" jsonschema:"minLength=1" jsonschema_description:"Exact device identifier accepted by Epson Download Center, such as its model or series name."`
	OS       string `json:"os" jsonschema:"pattern=^[A-Za-z0-9_-]+$" jsonschema_description:"Epson operating-system token used to select compatible downloads."`
	Region   string `json:"region,omitempty" jsonschema:"pattern=^[A-Z]{2}$,default=GB" jsonschema_description:"Two-letter uppercase Epson region code. Defaults to GB."`
	CTI      string `json:"cti" jsonschema:"pattern=^[0-9]+$" jsonschema_description:"Numeric content-type identifier. Exactly one returned download must match."`
	Language string `json:"language,omitempty" jsonschema:"pattern=^[a-z]{2}(-[A-Z]{2})?$,default=en" jsonschema_description:"Language code, optionally followed by an uppercase region, such as en or en-GB. Defaults to en."`
}

// Validate rejects device identifiers that cannot identify a vendor record.
func (config EpsonConfig) Validate() error {
	if !validText(config.DeviceID) {
		return errors.New("device_id must be nonempty text without surrounding whitespace or control characters")
	}
	return nil
}

// Epson requires exactly one matching CTI and uses the vendor's public file origin.
func Epson(ctx context.Context, client *http.Client, config EpsonConfig) (Release, error) {
	query := url.Values{"device_id": {config.DeviceID}, "os": {config.OS}, "region": {config.Region}, "language": {config.Language}}
	endpoint := &url.URL{Scheme: "https", Host: "download-center.epson.com", Path: "/api/v1/modules/", RawQuery: query.Encode()}
	type item struct {
		CTI     json.RawMessage `json:"cti"`
		URL     string          `json:"url"`
		Version string          `json:"version"`
	}
	var payload struct {
		Items []item `json:"items"`
	}
	if err := jsonMetadata(ctx, client, endpoint, hostURL(endpoint.Host), &payload); err != nil {
		return Release{}, fmt.Errorf("epson: %w", err)
	}
	if payload.Items == nil {
		return Release{}, errors.New("epson: metadata does not contain an items list")
	}
	var matches []item
	for _, candidate := range payload.Items {
		cti := string(candidate.CTI)
		if strings.HasPrefix(cti, `"`) {
			if err := json.Unmarshal(candidate.CTI, &cti); err != nil {
				return Release{}, errors.New("epson: invalid CTI in metadata")
			}
		}
		if cti == config.CTI {
			matches = append(matches, candidate)
		}
	}
	if len(matches) != 1 {
		return Release{}, fmt.Errorf("epson: expected one matching CTI, found %d", len(matches))
	}
	selected := matches[0]
	if !validText(selected.Version) {
		return Release{}, errors.New("epson: selected item has no valid version")
	}
	download, err := url.Parse(selected.URL)
	if err != nil || !httpsURL(download) || (download.Host != "download-center.epson.com" && download.Host != "download3.ebz.epson.net") {
		return Release{}, errors.New("epson: unexpected installer URL")
	}
	filename := path.Base(download.Path)
	if !validFilename(filename) || strings.HasSuffix(download.Path, "/") || path.Clean(download.Path) != download.Path {
		return Release{}, errors.New("epson: invalid installer path")
	}
	if download.Host == "download-center.epson.com" {
		if !strings.HasPrefix(download.Path, "/f/module/") {
			return Release{}, errors.New("epson: unexpected Download Center installer path")
		}
		download.Host = "d21ceiiri21o6e.cloudfront.net"
	}
	return Release{URL: download.String(), Filename: filename, Version: selected.Version}, nil
}
