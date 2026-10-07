package downloads

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/woodleighschool/stemma/plugin"
)

func TestBlackmagicDiscoveryAndReplay(t *testing.T) {
	var gets, posts int
	client := fixtureClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			gets++
			_, _ = fmt.Fprint(w, `{"downloads":[{"name":"Fusion Studio 21.1","urls":{"Windows":[{"downloadId":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}}]}`)
		case http.MethodPost:
			posts++
			if r.URL.Path != "/api/register/us/download/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
				t.Fatalf("unexpected path %s", r.URL.Path)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["product"] != "Fusion Studio" || body["platform"] != "Windows" {
				t.Fatalf("request=%v", body)
			}
			_, _ = fmt.Fprintf(w, "https://sw.blackmagicdesign.com/Fusion_21.1_Windows.zip?token=%d", posts)
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	})
	registry := plugin.New("test", "1")
	if err := Register(registry, client); err != nil {
		t.Fatal(err)
	}
	config := json.RawMessage(`{"product":"Fusion Studio","platform":"Windows","name_pattern":"^Fusion Studio (?P<version>[0-9.]+)$"}`)
	found, err := invoke(t, registry, "discover", "blackmagic", plugin.ResolveRequest[json.RawMessage]{Config: config})
	if err != nil || found.Version != "21.1" || !found.Immutable || posts != 0 {
		t.Fatalf("discovery=%+v, error=%v", found, err)
	}
	if strings.Contains(string(found.Observation), "url") || strings.Contains(string(found.Observation), "token") {
		t.Fatalf("ephemeral observation: %s", found.Observation)
	}
	for range 2 {
		got, err := invoke(t, registry, "run", "blackmagic", plugin.ResolveRequest[json.RawMessage]{Config: config, Observation: found.Observation})
		if err != nil || got.Download == nil || got.Download.Filename != "Fusion_21.1_Windows.zip" || string(got.Evidence["blackmagic"]) != `{"version":"21.1"}` {
			t.Fatalf("run=%+v, error=%v", got, err)
		}
	}
	if gets != 1 || posts != 2 {
		t.Fatalf("GETs=%d POSTs=%d", gets, posts)
	}
}

func TestBlackmagicValidatesBeforeNetwork(t *testing.T) {
	client := &http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("validation made a network request")
		return nil, nil
	})}
	registry := plugin.New("test", "1")
	if err := Register(registry, client); err != nil {
		t.Fatal(err)
	}
	for _, config := range []string{`{}`, `{"product":"Resolve","name_pattern":"["}`, `{"product":"Resolve","name_pattern":"Resolve [0-9]+"}`, `{"product":"Resolve","name_pattern":"(?P<version>[0-9.]+)","registration":{}}`, `{"product":"Resolve","name_pattern":"(?P<version>[0-9.]+)","registration":{"firstname":"Alex","lastname":"Example","email":"alex@example.test","phone":"123456","city":"Example City","country":"au"}}`, `{"product":"Resolve","name_pattern":"(?P<version>[0-9.]+)","major":21}`} {
		for _, method := range []string{"validate", "discover", "run"} {
			if _, err := invoke(t, registry, method, "blackmagic", plugin.ResolveRequest[json.RawMessage]{Config: json.RawMessage(config)}); err == nil {
				t.Fatalf("accepted %s: %s", method, config)
			}
		}
	}
	config := json.RawMessage(`{"product":"Resolve","name_pattern":"(?P<version>[0-9.]+)"}`)
	if _, err := invoke(t, registry, "validate", "blackmagic", plugin.ResolveRequest[json.RawMessage]{Config: config}); err != nil {
		t.Fatal(err)
	}
	for _, observation := range []string{`{}`, `{"download_id":"../escape","version":"21.1"}`, `{"download_id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","version":"21.1beta"}`} {
		if _, err := invoke(t, registry, "run", "blackmagic", plugin.ResolveRequest[json.RawMessage]{Config: config, Observation: json.RawMessage(observation)}); err == nil {
			t.Fatalf("accepted observation %s", observation)
		}
	}
}
