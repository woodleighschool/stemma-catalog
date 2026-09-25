package discovery

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func fixtureClient(t *testing.T, fixtures map[string]string) *http.Client {
	t.Helper()
	return &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		body, ok := fixtures[req.URL.String()]
		if !ok {
			t.Fatalf("unexpected request: %s", req.URL)
		}
		if req.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", req.Method)
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: req}, nil
	})}
}

func TestBlenderSelectsNumericStableRelease(t *testing.T) {
	client := fixtureClient(t, map[string]string{
		"https://download.blender.org/release/": `<html><body>
		<a href="Blender5.9/">Blender5.9/</a>
		<a href="Blender5.10/">Blender5.10/</a>
		<a href="Blender6.0/">Blender6.0/</a>
		<a href="Blender5.11-beta/">Blender5.11-beta/</a>
		<a href="https://elsewhere.example/Blender5.99/">Blender5.99/</a>
		<a href="Blender5.9/">Blender5.9/</a>
		</body></html>`,
		"https://download.blender.org/release/Blender5.10/": `<html><body>
		<a href="blender-5.10.9-macos-arm64.dmg">arm64</a>
		<a href="blender-5.10.11-macos-arm64.dmg">arm64</a>
		<a href="blender-5.10.12-macos-x64.dmg">x64</a>
		<a href="blender-5.10.13-beta-macos-arm64.dmg">beta</a>
		<a href="blender-5.10.12-macos-arm64.dmg.sha256">checksum</a>
		<a href="blender-5.10.11-macos-arm64.dmg">duplicate anchor</a>
		<a href="blender-5.10.9-macos-arm64.dmg">arm64</a>
		</body></html>`,
	})
	got, err := Blender(t.Context(), client, BlenderConfig{Major: 5, Architecture: ArchitectureARM64})
	if err != nil {
		t.Fatal(err)
	}
	want := Release{
		URL:      "https://download.blender.org/release/Blender5.10/blender-5.10.11-macos-arm64.dmg",
		Filename: "blender-5.10.11-macos-arm64.dmg",
		Version:  "5.10.11",
	}
	if got != want {
		t.Fatalf("release = %+v, want %+v", got, want)
	}
	got, err = Blender(t.Context(), client, BlenderConfig{Major: 5, Architecture: "x64"})
	if err != nil || got.Version != "5.10.12" {
		t.Fatalf("x64 release = %+v, error = %v", got, err)
	}
}

func TestBlenderRequiresStableInstallerInNewestDirectory(t *testing.T) {
	client := fixtureClient(t, map[string]string{
		"https://download.blender.org/release/":            `<a href="Blender5.1/">5.1</a><a href="Blender5.2/">5.2</a>`,
		"https://download.blender.org/release/Blender5.2/": `<a href="blender-5.2.0-beta-macos-arm64.dmg">beta</a>`,
	})
	if _, err := Blender(t.Context(), client, BlenderConfig{Major: 5, Architecture: ArchitectureARM64}); err == nil {
		t.Fatal("accepted release directory without a stable installer")
	}
}

func TestPythonSelectsLatestStablePatchOfExactBranch(t *testing.T) {
	client := fixtureClient(t, map[string]string{
		"https://www.python.org/downloads/macos/": `<html><ul>
		<li><a href="/downloads/release/python-3139/">Python 3.13.9 - May 1, 2026</a></li>
		<li><a href="/downloads/release/python-31312/">Python 3.13.12 - May 2, 2026</a></li>
		<li>Python 3.13.13rc1 - May 3, 2026</li>
		<li>Python 3.14.9 - May 4, 2026</li>
		<li>Python 3.130.99 - May 5, 2026</li>
		<li>Python 3.13.10 - May 6, 2026</li>
		</ul></html>`,
	})
	got, err := Python(t.Context(), client, PythonConfig{Branch: "3.13"})
	if err != nil {
		t.Fatal(err)
	}
	want := Release{
		URL:      "https://www.python.org/ftp/python/3.13.12/python-3.13.12-macos11.pkg",
		Filename: "python-3.13.12-macos11.pkg",
		Version:  "3.13.12",
	}
	if got != want {
		t.Fatalf("release = %+v, want %+v", got, want)
	}
	if _, err := Python(t.Context(), client, PythonConfig{Branch: "3.15"}); err == nil {
		t.Fatal("accepted missing branch")
	}
}

func TestCricutResolvesRolloutInstaller(t *testing.T) {
	filename := "Cricut Design Space-Install-v9.1.2.dmg"
	query := url.Values{"operatingSystem": {"osxnative"}, "shard": {"a"}, "fileName": {filename}}
	client := fixtureClient(t, map[string]string{
		"https://apis.cricut.com/desktopdownload/UpdateJson?operatingSystem=osxnative&shard=a": `{"result":"https://static.cricut.com/desktop/update.json"}`,
		"https://static.cricut.com/desktop/update.json":                                        `{"rolloutInstallFile":"` + filename + `","version":"unrelated updater version"}`,
		"https://apis.cricut.com/desktopdownload/InstallerFile?" + query.Encode():              `{"result":"https://static.cricut.com/desktop/` + url.PathEscape(filename) + `?token=a%2Fb"}`,
	})
	got, err := Cricut(t.Context(), client, CricutConfig{Shard: "a"})
	if err != nil {
		t.Fatal(err)
	}
	want := Release{URL: "https://static.cricut.com/desktop/" + url.PathEscape(filename) + "?token=a%2Fb", Filename: filename, Signed: true}
	if got != want {
		t.Fatalf("release = %+v, want %+v", got, want)
	}
}

func TestCricutRejectsInvalidMetadata(t *testing.T) {
	for _, test := range []struct {
		name     string
		update   string
		rollout  string
		artifact string
	}{
		{name: "missing update", update: `{}`},
		{name: "foreign update host", update: `{"result":"https://evil.example/update.json"}`},
		{name: "suffix lookalike", update: `{"result":"https://cricut.com.evil.example/update.json"}`},
		{name: "prefix lookalike", update: `{"result":"https://evilcricut.com/update.json"}`},
		{name: "userinfo", update: `{"result":"https://name@static.cricut.com/update.json"}`},
		{name: "port", update: `{"result":"https://static.cricut.com:443/update.json"}`},
		{name: "fragment", update: `{"result":"https://static.cricut.com/update.json#part"}`},
		{name: "plaintext", update: `{"result":"http://static.cricut.com/update.json"}`},
		{name: "bad JSON", update: `<html>PRIVATE ERROR BODY</html>`},
		{name: "missing rollout", rollout: `{}`},
		{name: "traversal", rollout: `{"rolloutInstallFile":"../Installer.dmg"}`},
		{name: "separator", rollout: `{"rolloutInstallFile":"path\\Installer.dmg"}`},
		{name: "non dmg", rollout: `{"rolloutInstallFile":"Installer.exe"}`},
		{name: "foreign artifact", artifact: `{"result":"https://evil.example/Installer.dmg"}`},
		{name: "different artifact", artifact: `{"result":"https://static.cricut.com/Other.dmg"}`},
		{name: "artifact traversal", artifact: `{"result":"https://static.cricut.com/a/%2e%2e/Installer.dmg"}`},
		{name: "artifact trailing slash", artifact: `{"result":"https://static.cricut.com/Installer.dmg/"}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.update == "" {
				test.update = `{"result":"https://static.cricut.com/update.json"}`
			}
			if test.rollout == "" {
				test.rollout = `{"rolloutInstallFile":"Installer.dmg"}`
			}
			client := fixtureClient(t, map[string]string{
				"https://apis.cricut.com/desktopdownload/UpdateJson?operatingSystem=osxnative&shard=a":                           test.update,
				"https://static.cricut.com/update.json":                                                                          test.rollout,
				"https://apis.cricut.com/desktopdownload/InstallerFile?fileName=Installer.dmg&operatingSystem=osxnative&shard=a": test.artifact,
			})
			_, err := Cricut(t.Context(), client, CricutConfig{Shard: "a"})
			if err == nil {
				t.Fatal("accepted invalid metadata")
			}
			if strings.Contains(err.Error(), "PRIVATE ERROR BODY") {
				t.Fatalf("error leaked response body: %v", err)
			}
		})
	}
}

func TestEpsonPreservesVersionAndRewritesPublicOrigin(t *testing.T) {
	for _, test := range []struct {
		name string
		cti  string
		url  string
		want string
	}{
		{"numeric CTI", `2001`, "https://download-center.epson.com/f/module/aa/Epson%20Driver.dmg?download=1", "https://d21ceiiri21o6e.cloudfront.net/f/module/aa/Epson%20Driver.dmg?download=1"},
		{"string CTI", `"2001"`, "https://download3.ebz.epson.net/dsc/f/03/Epson%20Driver.dmg", "https://download3.ebz.epson.net/dsc/f/03/Epson%20Driver.dmg"},
	} {
		t.Run(test.name, func(t *testing.T) {
			device := "AM-C6000 Series & Special"
			query := url.Values{"device_id": {device}, "os": {"MAC26"}, "region": {"GB"}, "language": {"en"}}
			client := fixtureClient(t, map[string]string{
				"https://download-center.epson.com/api/v1/modules/?" + query.Encode(): `{"items":[{"cti":1000,"version":"99.0","url":"https://example.invalid/other.dmg"},{"cti":` + test.cti + `,"version":"13.04.00","url":"` + test.url + `"}]}`,
			})
			got, err := Epson(t.Context(), client, EpsonConfig{DeviceID: device, OS: "MAC26", CTI: "2001", Region: "GB", Language: "en"})
			if err != nil {
				t.Fatal(err)
			}
			want := Release{URL: test.want, Filename: "Epson Driver.dmg", Version: "13.04.00"}
			if got != want {
				t.Fatalf("release = %+v, want %+v", got, want)
			}
		})
	}
}

func TestEpsonRejectsAmbiguousOrMalformedMetadata(t *testing.T) {
	for _, test := range []struct {
		name    string
		payload string
	}{
		{"missing items", `{}`},
		{"no match", `{"items":[]}`},
		{"ambiguous", `{"items":[{"cti":2001},{"cti":"2001"}]}`},
		{"nonobject", `[]`},
		{"missing version", `{"items":[{"cti":2001,"url":"https://download-center.epson.com/f/module/file.dmg"}]}`},
		{"invalid version", `{"items":[{"cti":2001,"version":"13.4\n","url":"https://download-center.epson.com/f/module/file.dmg"}]}`},
		{"missing URL", `{"items":[{"cti":2001,"version":"13.4"}]}`},
		{"foreign host", `{"items":[{"cti":2001,"version":"13.4","url":"https://evil.example/f/module/file.dmg"}]}`},
		{"direct cloudfront", `{"items":[{"cti":2001,"version":"13.4","url":"https://d21ceiiri21o6e.cloudfront.net/f/module/file.dmg"}]}`},
		{"plaintext", `{"items":[{"cti":2001,"version":"13.4","url":"http://download-center.epson.com/f/module/file.dmg"}]}`},
		{"userinfo", `{"items":[{"cti":2001,"version":"13.4","url":"https://name@download-center.epson.com/f/module/file.dmg"}]}`},
		{"port", `{"items":[{"cti":2001,"version":"13.4","url":"https://download-center.epson.com:443/f/module/file.dmg"}]}`},
		{"fragment", `{"items":[{"cti":2001,"version":"13.4","url":"https://download-center.epson.com/f/module/file.dmg#part"}]}`},
		{"unexpected path", `{"items":[{"cti":2001,"version":"13.4","url":"https://download-center.epson.com/download/file.dmg"}]}`},
		{"traversal", `{"items":[{"cti":2001,"version":"13.4","url":"https://download-center.epson.com/f/module/%2e%2e/file.dmg"}]}`},
		{"directory", `{"items":[{"cti":2001,"version":"13.4","url":"https://download-center.epson.com/f/module/files/"}]}`},
		{"unsafe filename", `{"items":[{"cti":2001,"version":"13.4","url":"https://download-center.epson.com/f/module/file%5cname.dmg"}]}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := fixtureClient(t, map[string]string{
				"https://download-center.epson.com/api/v1/modules/?device_id=AM-C6000+Series&language=en&os=MAC26&region=GB": test.payload,
			})
			if _, err := Epson(t.Context(), client, EpsonConfig{DeviceID: "AM-C6000 Series", OS: "MAC26", CTI: "2001", Region: "GB", Language: "en"}); err == nil {
				t.Fatal("accepted ambiguous or malformed metadata")
			}
		})
	}
}

func TestMetadataBoundsAndHTTPFailures(t *testing.T) {
	for _, test := range []struct {
		name   string
		status int
		body   string
	}{
		{"unauthorized", http.StatusUnauthorized, "PRIVATE ERROR BODY"},
		{"empty response", http.StatusNoContent, ""},
		{"oversized", http.StatusOK, strings.Repeat("x", metadataLimit+1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: test.status, Body: io.NopCloser(strings.NewReader(test.body)), Header: make(http.Header), Request: req}, nil
			})}
			_, err := Python(t.Context(), client, PythonConfig{Branch: "3.13"})
			if err == nil {
				t.Fatal("accepted failed metadata response")
			}
			if strings.Contains(err.Error(), "PRIVATE ERROR BODY") {
				t.Fatalf("error leaked body: %v", err)
			}
		})
	}
}

func TestMetadataRejectsForeignRedirect(t *testing.T) {
	requests := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		return &http.Response{StatusCode: http.StatusFound, Body: http.NoBody, Header: http.Header{"Location": {"https://elsewhere.example/metadata"}}, Request: req}, nil
	})}
	if _, err := Python(t.Context(), client, PythonConfig{Branch: "3.13"}); err == nil {
		t.Fatal("accepted foreign redirect")
	}
	if requests != 1 {
		t.Fatalf("requested redirect target: %d requests", requests)
	}
}

func TestMetadataPropagatesCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Context().Err() == nil {
			return nil, fmt.Errorf("request context was not cancelled")
		}
		return nil, req.Context().Err()
	})}
	_, err := Python(ctx, client, PythonConfig{Branch: "3.13"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want cancellation", err)
	}
}
