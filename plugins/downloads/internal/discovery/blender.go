package discovery

import (
	"context"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// BlenderConfig selects a macOS architecture within a Blender major release.
type BlenderConfig struct {
	Major        int    `json:"major"`
	Architecture string `json:"architecture"`
}

var hrefPattern = regexp.MustCompile(`(?i)\bhref\s*=\s*["']([^"']+)["']`)

// Blender selects the latest stable patch in the newest minor release directory.
func Blender(ctx context.Context, client *http.Client, config BlenderConfig) (Release, error) {
	if config.Major < 1 {
		return Release{}, errors.New("blender: major must be positive")
	}
	if config.Architecture == "" {
		config.Architecture = "arm64"
	}
	if config.Architecture != "arm64" && config.Architecture != "x64" {
		return Release{}, errors.New("blender: architecture must be arm64 or x64")
	}
	base := &url.URL{Scheme: "https", Host: "download.blender.org", Path: "/release/"}
	data, err := metadata(ctx, client, base, hostURL(base.Host))
	if err != nil {
		return Release{}, fmt.Errorf("blender: %w", err)
	}
	directoryPattern := regexp.MustCompile(fmt.Sprintf(`^Blender%d\.(0|[1-9][0-9]*)/$`, config.Major))
	minor := -1
	for _, name := range listingNames(data, base) {
		match := directoryPattern.FindStringSubmatch(name)
		if match == nil {
			continue
		}
		value, err := strconv.Atoi(match[1])
		if err == nil && value > minor {
			minor = value
		}
	}
	if minor < 0 {
		return Release{}, errors.New("blender: no stable release directory matches major")
	}
	base.Path += fmt.Sprintf("Blender%d.%d/", config.Major, minor)
	data, err = metadata(ctx, client, base, hostURL(base.Host))
	if err != nil {
		return Release{}, fmt.Errorf("blender: %w", err)
	}
	installerPattern := regexp.MustCompile(fmt.Sprintf(`^blender-(%d\.%d\.(0|[1-9][0-9]*))-macos-%s\.dmg$`, config.Major, minor, config.Architecture))
	patch := -1
	var release Release
	for _, name := range listingNames(data, base) {
		match := installerPattern.FindStringSubmatch(name)
		if match == nil {
			continue
		}
		value, err := strconv.Atoi(match[2])
		if err == nil && value > patch {
			patch = value
			release = Release{URL: base.String() + name, Filename: name, Version: match[1]}
		}
	}
	if patch < 0 {
		return Release{}, errors.New("blender: no stable macOS installer matches architecture")
	}
	return release, nil
}

func listingNames(data []byte, base *url.URL) []string {
	var names []string
	for _, match := range hrefPattern.FindAllSubmatch(data, -1) {
		reference, err := url.Parse(html.UnescapeString(string(match[1])))
		if err != nil {
			continue
		}
		target := base.ResolveReference(reference)
		if !hostURL(base.Host)(target) || target.RawQuery != "" || !strings.HasPrefix(target.Path, base.Path) {
			continue
		}
		names = append(names, strings.TrimPrefix(target.Path, base.Path))
	}
	return names
}
