package downloads

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/woodleighschool/stemma-catalog/plugins/downloads/internal/discovery"
	"github.com/woodleighschool/stemma/plugin"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func fixtureClient(t *testing.T, handler http.HandlerFunc) *http.Client {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)
	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	transport := server.Client().Transport
	client := Client()
	client.Transport = transportFunc(func(request *http.Request) (*http.Response, error) {
		copy := request.Clone(request.Context())
		copy.URL.Scheme, copy.URL.Host = target.Scheme, target.Host
		return transport.RoundTrip(copy)
	})
	return client
}

func invoke(t *testing.T, registry *plugin.Registry, method, operation string, request plugin.ResolveRequest[json.RawMessage]) (plugin.ResolveResponse, error) {
	t.Helper()
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	response, err := registry.Handle(t.Context(), plugin.Request{Protocol: plugin.ProtocolVersion, Method: method, Operation: operation, Input: data})
	var result plugin.ResolveResponse
	if err == nil && len(response.Output) > 0 {
		err = json.Unmarshal(response.Output, &result)
	}
	return result, err
}

func TestRunFetchesOnlyTheObservedArtifact(t *testing.T) {
	const body = "synthetic installer bytes"
	client := fixtureClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/releases/old.pkg" {
			t.Errorf("run rediscovered %s", r.URL.Path)
			http.Error(w, "unexpected discovery", http.StatusGone)
			return
		}
		_, _ = fmt.Fprint(w, body)
	})
	registry := plugin.New("test", "1")
	if err := Register(registry, client); err != nil {
		t.Fatal(err)
	}
	configs := map[string]string{
		"audinate":  `{"product":"dante-controller"}`,
		"microsoft": `{"product":"outlook"}`,
		"blender":   `{"major":5}`,
		"python":    `{"branch":"3.13"}`,
		"cricut":    `{}`,
		"epson":     `{"device_id":"AM-C6000 Series","os":"MAC26","cti":"2001"}`,
	}
	observation := json.RawMessage(`{"url":"https://downloads.example.test/releases/old.pkg","filename":"old.pkg","version":"1.2.3"}`)
	hash := sha256.Sum256([]byte(body))
	for operation, config := range configs {
		t.Run(operation, func(t *testing.T) {
			request := plugin.ResolveRequest[json.RawMessage]{Config: json.RawMessage(config), Root: t.TempDir(), Workspace: t.TempDir(), Observation: observation}
			result, err := invoke(t, registry, "run", operation, request)
			if err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(result.Artifact.Path)
			if err != nil || string(data) != body || result.Artifact.SHA256 != hex.EncodeToString(hash[:]) || result.Artifact.Size != int64(len(body)) {
				t.Fatalf("artifact=%+v, content=%q, error=%v", result.Artifact, data, err)
			}
			if result.Artifact.Path != filepath.Join(request.Workspace, "old.pkg") || string(result.Artifact.Evidence[operation]) != `{"version":"1.2.3"}` {
				t.Fatalf("artifact=%+v", result.Artifact)
			}
		})
	}
}

func TestSignedDownloadsAreDiscoveredAgainRatherThanRecorded(t *testing.T) {
	const filename = "CricutDesignSpace-Install-v9.1.2.dmg"
	var signed atomic.Int32
	client := fixtureClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/desktopdownload/UpdateJson":
			_, _ = fmt.Fprint(w, `{"result":"https://static.cricut.com/desktop/update.json"}`)
		case "/desktop/update.json":
			_, _ = fmt.Fprintf(w, `{"rolloutInstallFile":%q}`, filename)
		case "/desktopdownload/InstallerFile":
			_, _ = fmt.Fprintf(w, `{"result":"https://static.cricut.com/desktop/%s?token=%d"}`, filename, signed.Add(1))
		case "/desktop/" + filename:
			if r.URL.Query().Get("token") != strconv.Itoa(int(signed.Load())) {
				http.Error(w, "expired", http.StatusForbidden)
				return
			}
			_, _ = fmt.Fprint(w, "installer")
		default:
			http.NotFound(w, r)
		}
	})
	registry := plugin.New("test", "1")
	if err := Register(registry, client); err != nil {
		t.Fatal(err)
	}
	found, err := invoke(t, registry, "discover", "cricut", plugin.ResolveRequest[json.RawMessage]{Config: json.RawMessage(`{}`)})
	if err != nil || found.Immutable || strings.Contains(string(found.Observation), "token") || !strings.Contains(string(found.Observation), filename) {
		t.Fatalf("discover: %v observation=%s immutable=%v", err, found.Observation, found.Immutable)
	}
	fetched, err := invoke(t, registry, "run", "cricut", plugin.ResolveRequest[json.RawMessage]{Config: json.RawMessage(`{}`), Workspace: t.TempDir(), Observation: found.Observation})
	if err != nil || fetched.Artifact.Filename != filename || signed.Load() != 2 {
		t.Fatalf("run: %v artifact=%+v signed=%d", err, fetched.Artifact, signed.Load())
	}
}

func TestValidationRejectsInvalidConfigurationWithoutNetwork(t *testing.T) {
	client := &http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("validation performed HTTP request")
		return nil, nil
	})}
	registry := plugin.New("test", "1")
	if err := Register(registry, client); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ operation, config string }{
		{"audinate", `{}`},
		{"audinate", `{"product":"unknown"}`},
		{"microsoft", `{}`},
		{"microsoft", `{"product":"unknown"}`},
		{"microsoft", `{"product":"outlook","channel":"unknown"}`},
		{"microsoft", `{"product":"outlook","type":"delta"}`},
		{"microsoft", `{"product":"outlook","url":"https://example.test"}`},
		{"microsoft", `{"product":"edge","channel":"beta"}`},
		{"microsoft", `{"product":"office","type":"updater"}`},
		{"blender", `{"major":0}`},
		{"blender", `{"major":5,"url":"https://example.test"}`},
		{"python", `{"branch":"3.13rc1"}`},
		{"python", `{"branch":"2.7"}`},
		{"python", `{"branch":"3.9"}`},
		{"epson", `{"device_id":" ","os":"MAC26","cti":"2001"}`},
		{"epson", `{"device_id":"Printer","os":"MAC 26","cti":"2001"}`},
		{"epson", `{"device_id":"Printer","os":"MAC26","cti":"2001","region":"gb"}`},
		{"cricut", `{"operating_system":"windows"}`},
		{"epson", `{"device_id":"Printer","os":"MAC26","cti":2001}`},
	} {
		for _, method := range []string{"validate", "discover", "run"} {
			if _, err := invoke(t, registry, method, test.operation, plugin.ResolveRequest[json.RawMessage]{Config: json.RawMessage(test.config)}); err == nil {
				t.Fatalf("accepted %s %s on %s", test.operation, test.config, method)
			}
		}
	}
	for _, config := range []string{`{"product":"outlook"}`, `{"product":"outlook","channel":"preview","type":"updater"}`, `{"product":"edge"}`} {
		if _, err := invoke(t, registry, "validate", "microsoft", plugin.ResolveRequest[json.RawMessage]{Config: json.RawMessage(config)}); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := invoke(t, registry, "validate", "python", plugin.ResolveRequest[json.RawMessage]{Config: json.RawMessage(`{"branch":"3.13"}`)}); err != nil {
		t.Fatal(err)
	}
}

func TestDownloadRejectsUnsafeObservationAndRedirect(t *testing.T) {
	for _, release := range []discovery.Release{
		{URL: "http://example.test/app.pkg", Filename: "app.pkg"},
		{URL: "https://secret@example.test/app.pkg", Filename: "app.pkg"},
		{URL: "https://example.test/app.pkg#secret", Filename: "app.pkg"},
		{URL: "https://example.test/app.pkg", Filename: "../app.pkg"},
		{URL: "https://example.test/app.pkg", Filename: `..\app.pkg`},
	} {
		if _, err := download(t.Context(), Client(), release, t.TempDir()); err == nil {
			t.Fatalf("accepted %+v", release)
		}
	}
	client := fixtureClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://insecure.example.test/app.pkg", http.StatusFound)
	})
	if _, err := download(t.Context(), client, discovery.Release{URL: "https://example.test/app.pkg", Filename: "app.pkg"}, t.TempDir()); err == nil {
		t.Fatal("accepted insecure redirect")
	}
}

func TestFailedDownloadLeavesNoArtifact(t *testing.T) {
	for _, status := range []int{http.StatusOK, http.StatusForbidden} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			client := fixtureClient(t, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Length", "100")
				w.WriteHeader(status)
				_, _ = fmt.Fprint(w, "private error response")
			})
			workspace := t.TempDir()
			_, err := download(t.Context(), client, discovery.Release{URL: "https://example.test/app.pkg", Filename: "app.pkg"}, workspace)
			if err == nil || strings.Contains(err.Error(), "private error response") {
				t.Fatalf("download error: %v", err)
			}
			entries, err := os.ReadDir(workspace)
			if err != nil || len(entries) != 0 {
				t.Fatalf("partial files: %v, %v", entries, err)
			}
		})
	}
}

func TestDownloadReportsTransferProgress(t *testing.T) {
	body := strings.Repeat("x", 4096)
	client := fixtureClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		_, _ = fmt.Fprint(w, body)
	})
	var logs bytes.Buffer
	ctx := plugin.WithLogger(t.Context(), slog.New(slog.NewJSONHandler(&logs, nil)))
	if _, err := download(ctx, client, discovery.Release{URL: "https://example.test/app.pkg", Filename: "app.pkg"}, t.TempDir()); err != nil {
		t.Fatal(err)
	}
	var last struct {
		Progress       bool
		Current, Total int64
	}
	for line := range strings.Lines(logs.String()) {
		if err := json.Unmarshal([]byte(line), &last); err != nil {
			t.Fatal(err)
		}
	}
	if !last.Progress || last.Current != int64(len(body)) || last.Total != int64(len(body)) {
		t.Fatalf("final progress %+v from %s", last, logs.String())
	}
}

func TestDownloadCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := download(ctx, Client(), discovery.Release{URL: "https://example.test/app.pkg", Filename: "app.pkg"}, t.TempDir())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
}
