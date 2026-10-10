package discovery

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/invopop/jsonschema"
)

// UnityConfig selects an Editor release line and the stream it belongs to.
// Stream defaults to lts.
type UnityConfig struct {
	Line   string      `json:"line" jsonschema:"pattern=^[0-9]+\\.[0-9]+$" jsonschema_description:"Editor release line as major.minor, such as 6000.3. Releases of other lines are never selected."`
	Stream UnityStream `json:"stream,omitempty" jsonschema:"default=lts" jsonschema_description:"Release stream the line's releases must belong to. Long-term support lines use lts and update release lines use supported."`
}

// UnityStream names a Unity release stream.
type UnityStream string

const (
	UnityLTS       UnityStream = "lts"
	UnitySupported UnityStream = "supported"
	UnityBeta      UnityStream = "beta"
	UnityAlpha     UnityStream = "alpha"
)

func (UnityStream) JSONSchemaExtend(schema *jsonschema.Schema) {
	schema.Enum = []any{UnityLTS, UnitySupported, UnityBeta, UnityAlpha}
}

// Unity selects the newest Apple silicon macOS Editor installer of a release line.
func Unity(ctx context.Context, client *http.Client, config UnityConfig) (Release, error) {
	// The feed filters versions by text prefix; the dot keeps 6000.3 from matching 6000.30.
	line := config.Line + "."
	query := url.Values{
		"architecture": {"ARM64"},
		"limit":        {"1"},
		"order":        {"RELEASE_DATE_DESC"},
		"platform":     {"MAC_OS"},
		"stream":       {strings.ToUpper(string(config.Stream))},
		"version":      {line},
	}
	endpoint := &url.URL{Scheme: "https", Host: "services.api.unity.com", Path: "/unity/editor/release/v1/releases", RawQuery: query.Encode()}
	type download struct {
		URL          string `json:"url"`
		Type         string `json:"type"`
		Platform     string `json:"platform"`
		Architecture string `json:"architecture"`
	}
	var payload struct {
		Results []struct {
			Version   string     `json:"version"`
			Downloads []download `json:"downloads"`
		} `json:"results"`
	}
	if err := jsonMetadata(ctx, client, endpoint, hostURL(endpoint.Host), &payload); err != nil {
		return Release{}, fmt.Errorf("unity: %w", err)
	}
	if len(payload.Results) == 0 {
		return Release{}, fmt.Errorf("unity: no %s release in line %s", config.Stream, config.Line)
	}
	if len(payload.Results) != 1 {
		return Release{}, errors.New("unity: metadata lists more than the newest release")
	}
	release := payload.Results[0]
	if !validText(release.Version) || !strings.HasPrefix(release.Version, line) {
		return Release{}, errors.New("unity: selected release is outside the line")
	}
	var installers []download
	for _, candidate := range release.Downloads {
		if candidate.Platform == "MAC_OS" && candidate.Architecture == "ARM64" && candidate.Type == "PKG" {
			installers = append(installers, candidate)
		}
	}
	if len(installers) != 1 {
		return Release{}, fmt.Errorf("unity: expected one Apple silicon macOS installer, found %d", len(installers))
	}
	installer, err := url.Parse(installers[0].URL)
	if err != nil || !httpsURL(installer) || installer.Host != "download.unity3d.com" {
		return Release{}, errors.New("unity: unexpected installer URL")
	}
	filename := path.Base(installer.Path)
	if !validFilename(filename) || !strings.HasSuffix(filename, ".pkg") || path.Clean(installer.Path) != installer.Path {
		return Release{}, errors.New("unity: invalid installer path")
	}
	return Release{URL: installer.String(), Filename: filename, Version: release.Version}, nil
}
