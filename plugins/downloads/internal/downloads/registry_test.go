package downloads

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
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
		clone := request.Clone(request.Context())
		clone.URL.Scheme, clone.URL.Host = target.Scheme, target.Host
		return transport.RoundTrip(clone)
	})
	return client
}

func invoke(t *testing.T, registry *plugin.Registry, method, operation string, request plugin.ResolveRequest[json.RawMessage]) (plugin.ResolveResponse, error) {
	t.Helper()
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	response, err := registry.Handle(t.Context(), plugin.Request{Method: method, Operation: operation, Input: data})
	var result plugin.ResolveResponse
	if err == nil && len(response.Output) > 0 {
		err = json.Unmarshal(response.Output, &result)
	}
	return result, err
}

func TestRunReturnsOnlyTheObservedDownload(t *testing.T) {
	client := &http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("run made a network request for a recorded URL")
		return nil, nil
	})}
	registry := plugin.New("test", "1")
	if err := Register(registry, client); err != nil {
		t.Fatal(err)
	}
	configs := map[string]string{
		"microsoft": `{"product":"outlook"}`,
		"python":    `{"branch":"3.13"}`,
		"epson":     `{"device_id":"AM-C6000 Series","os":"MAC26","cti":"2001"}`,
	}
	observation := json.RawMessage(`{"url":"https://downloads.example.test/releases/old.pkg","filename":"old.pkg","version":"1.2.3"}`)
	for operation, config := range configs {
		t.Run(operation, func(t *testing.T) {
			request := plugin.ResolveRequest[json.RawMessage]{Config: json.RawMessage(config), Observation: observation}
			result, err := invoke(t, registry, "run", operation, request)
			if err != nil {
				t.Fatal(err)
			}
			if result.Download == nil || result.Download.URL != "https://downloads.example.test/releases/old.pkg" || result.Download.Filename != "old.pkg" {
				t.Fatalf("download=%+v", result.Download)
			}
			if result.Artifact.Path != "" || string(result.Evidence[operation]) != `{"version":"1.2.3"}` {
				t.Fatalf("response=%+v", result)
			}
		})
	}
}

func TestSignedReplayKeepsTheLockedInstallerAfterTheRolloutAdvances(t *testing.T) {
	const filename = "CricutDesignSpace-Install-v9.1.2.dmg"
	var signed, rollouts atomic.Int32
	var advanced atomic.Bool
	client := fixtureClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/desktopdownload/UpdateJson":
			_, _ = fmt.Fprint(w, `{"result":"https://static.cricut.com/desktop/update.json"}`)
		case "/desktop/update.json":
			rollouts.Add(1)
			current := filename
			if advanced.Load() {
				current = "CricutDesignSpace-Install-v9.2.0.dmg"
			}
			_, _ = fmt.Fprintf(w, `{"rolloutInstallFile":%q}`, current)
		case "/desktopdownload/InstallerFile":
			if r.URL.Query().Get("operatingSystem") != "osxnative" || r.URL.Query().Get("shard") != "a" {
				t.Errorf("unexpected installer selection %s", r.URL.RawQuery)
			}
			_, _ = fmt.Fprintf(w, `{"result":"https://static.cricut.com/desktop/%s?token=%d"}`, r.URL.Query().Get("fileName"), signed.Add(1))
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
			http.NotFound(w, r)
		}
	})
	registry := plugin.New("test", "1")
	if err := Register(registry, client); err != nil {
		t.Fatal(err)
	}
	found, err := invoke(t, registry, "discover", "cricut", plugin.ResolveRequest[json.RawMessage]{Config: json.RawMessage(`{}`)})
	if err != nil || found.Immutable || found.Download != nil || strings.Contains(string(found.Observation), "token") || !strings.Contains(string(found.Observation), filename) {
		t.Fatalf("discover: %v observation=%s immutable=%v", err, found.Observation, found.Immutable)
	}
	advanced.Store(true)
	fetched, err := invoke(t, registry, "run", "cricut", plugin.ResolveRequest[json.RawMessage]{Config: json.RawMessage(`{}`), Observation: found.Observation})
	if err != nil || fetched.Download == nil || fetched.Download.Filename != filename || fetched.Download.URL != "https://static.cricut.com/desktop/"+filename+"?token=2" || signed.Load() != 2 || rollouts.Load() != 1 {
		t.Fatalf("run: %v download=%+v signed=%d rollouts=%d", err, fetched.Download, signed.Load(), rollouts.Load())
	}
}

func TestSignedReplayRejectsADifferentInstaller(t *testing.T) {
	client := fixtureClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/desktopdownload/InstallerFile" || r.URL.Query().Get("fileName") != "old.dmg" {
			t.Errorf("unexpected request %s", r.URL)
		}
		_, _ = fmt.Fprint(w, `{"result":"https://static.cricut.com/new.dmg?token=secret"}`)
	})
	registry := plugin.New("test", "1")
	if err := Register(registry, client); err != nil {
		t.Fatal(err)
	}
	_, err := invoke(t, registry, "run", "cricut", plugin.ResolveRequest[json.RawMessage]{Config: json.RawMessage(`{}`), Observation: json.RawMessage(`{"filename":"old.dmg","version":""}`)})
	if err == nil || !strings.Contains(err.Error(), "does not match recorded filename") || strings.Contains(err.Error(), "secret") {
		t.Fatalf("changed installer: %v", err)
	}
}

func TestSignedReplayRejectsInvalidObservationsBeforeRequestingAURL(t *testing.T) {
	client := &http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("invalid observation made a network request")
		return nil, nil
	})}
	registry := plugin.New("test", "1")
	if err := Register(registry, client); err != nil {
		t.Fatal(err)
	}
	for _, observation := range []string{
		`{}`,
		`{"filename":"../old.dmg"}`,
		`{"filename":"old.pkg"}`,
		`{"filename":"old.dmg","version":"1"}`,
		`{"filename":"old.dmg","url":"https://static.cricut.com/old.dmg?token=secret"}`,
	} {
		_, err := invoke(t, registry, "run", "cricut", plugin.ResolveRequest[json.RawMessage]{Config: json.RawMessage(`{}`), Observation: json.RawMessage(observation)})
		if err == nil || !strings.Contains(err.Error(), "invalid installer observation") {
			t.Fatalf("observation %s: %v", observation, err)
		}
	}
}

func TestDiscoveryRecordsVersionWithoutDownloadInstructions(t *testing.T) {
	resolve := configured(Client(), "fixture", func(context.Context, *http.Client, struct{}) (discovery.Release, error) {
		return discovery.Release{URL: "https://example.test/app.pkg", Filename: "app.pkg", Version: "1.2.3"}, nil
	}, nil)
	result, err := resolve(t.Context(), plugin.ResolveRequest[struct{}]{Method: "discover"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Version != "1.2.3" || !result.Immutable || result.Download != nil || result.Artifact.Path != "" {
		t.Fatalf("discovery=%+v", result)
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
		{"microsoft", `{}`},
		{"microsoft", `{"product":"unknown"}`},
		{"microsoft", `{"product":"outlook","channel":"unknown"}`},
		{"microsoft", `{"product":"outlook","type":"delta"}`},
		{"microsoft", `{"product":"outlook","url":"https://example.test"}`},
		{"microsoft", `{"product":"edge","channel":"beta"}`},
		{"microsoft", `{"product":"office","type":"updater"}`},
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

func TestRunRejectsUnsafeObservations(t *testing.T) {
	resolve := configured(Client(), "fixture", func(context.Context, *http.Client, struct{}) (discovery.Release, error) {
		t.Fatal("run rediscovered an unsafe observation")
		return discovery.Release{}, nil
	}, nil)
	for _, release := range []discovery.Release{
		{Filename: "app.pkg"},
		{URL: "http://example.test/app.pkg", Filename: "app.pkg"},
		{URL: "https://secret@example.test/app.pkg", Filename: "app.pkg"},
		{URL: "https://example.test/app.pkg#secret", Filename: "app.pkg"},
		{URL: "https://example.test/app.pkg", Filename: "../app.pkg"},
		{URL: "https://example.test/app.pkg", Filename: `..\app.pkg`},
		{URL: "https://example.test/app.pkg", Filename: "app.pkg", Version: "1\n2"},
	} {
		observation, err := json.Marshal(release)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := resolve(t.Context(), plugin.ResolveRequest[struct{}]{Method: "run", Observation: observation}); err == nil {
			t.Fatalf("accepted %+v", release)
		}
	}
}
