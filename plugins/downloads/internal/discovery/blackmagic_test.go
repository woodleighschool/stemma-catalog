package discovery

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

const blackmagicFeedURL = "https://www.blackmagicdesign.com/api/support/us/downloads.json"

func blackmagicConfig() BlackmagicConfig {
	return BlackmagicConfig{Product: "DaVinci Resolve", Platform: "Mac OS X", NamePattern: `^DaVinci Resolve (?P<version>[0-9]+(?:\.[0-9]+)*)(?: Update)?$`}
}

func TestBlackmagicSelectsNumericLatestAcrossMajors(t *testing.T) {
	entries := []string{
		`{"name":"DaVinci Resolve 20.9.9","urls":{"Mac OS X":[{"downloadId":"11111111111111111111111111111111"}]}}`,
		`{"name":"DaVinci Resolve Studio 99.0","urls":{"Mac OS X":[{"downloadId":"22222222222222222222222222222222"}]}}`,
		`{"name":"DaVinci Resolve 22 Public Beta","urls":{"Mac OS X":[{"downloadId":"33333333333333333333333333333333"}]}}`,
		`{"name":"DaVinci Resolve 22.0 Beta 2","urls":{"Mac OS X":[{"downloadId":"33333333333333333333333333333333"}]}}`,
		`{"name":"DaVinci Resolve 21.1.9 Update","urls":{"Mac OS X":[{"downloadId":"44444444444444444444444444444444"}]}}`,
		`{"name":"DaVinci Resolve 21.1.10 Update","requiresRegistration":true,"urls":{"Mac OS X":[{"downloadId":"55555555555555555555555555555555"}]}}`,
		`{"name":"DaVinci Resolve 23.0","urls":{"Windows":[{"downloadId":"66666666666666666666666666666666"}]}}`,
	}
	for _, reverse := range []bool{false, true} {
		if reverse {
			for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
		client := fixtureClient(t, map[string]string{blackmagicFeedURL: `{"downloads":[` + strings.Join(entries, ",") + `]}`})
		got, err := Blackmagic(t.Context(), client, blackmagicConfig())
		if err != nil || got.Version != "21.1.10" || got.DownloadID != "55555555555555555555555555555555" || !got.RequiresRegistration {
			t.Fatalf("release=%+v, error=%v", got, err)
		}
	}
}

func TestBlackmagicProductAndPlatformAreConfigured(t *testing.T) {
	config := BlackmagicConfig{Product: "Fusion Studio", Platform: "Windows", NamePattern: `^Fusion Studio (?P<version>[0-9.]+)$`}
	client := fixtureClient(t, map[string]string{blackmagicFeedURL: `{"downloads":[{"name":"Fusion Studio 21.2","urls":{"Mac OS X":[{"downloadId":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}],"Windows":[{"downloadId":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}]}},{"name":"DaVinci Resolve 25","urls":{"Windows":[{"downloadId":"cccccccccccccccccccccccccccccccc"}]}}]}`})
	got, err := Blackmagic(t.Context(), client, config)
	if err != nil || got.DownloadID != "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" || got.Version != "21.2" || got.RequiresRegistration {
		t.Fatalf("release=%+v, error=%v", got, err)
	}
}

func TestBlackmagicRejectsUnusableFeed(t *testing.T) {
	for _, feed := range []string{
		`{}`, `not JSON`,
		`{"downloads":[{"name":"DaVinci Resolve 21.1","urls":{"Mac OS X":[{"downloadId":"../escape"}]}}]}`,
		`{"downloads":[{"name":"DaVinci Resolve 21.1","urls":{"Mac OS X":[{"downloadId":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},{"downloadId":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}]}}]}`,
		`{"downloads":[{"name":"DaVinci Resolve 21.1","urls":{"Mac OS X":[{"downloadId":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}]}},{"name":"DaVinci Resolve 21.1 Update","urls":{"Mac OS X":[{"downloadId":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}]}}]}`,
	} {
		if _, err := Blackmagic(t.Context(), fixtureClient(t, map[string]string{blackmagicFeedURL: feed}), blackmagicConfig()); err == nil {
			t.Fatalf("accepted feed %s", feed)
		}
	}
}

func TestBlackmagicRegistrationUsesRecordedID(t *testing.T) {
	for _, test := range []struct{ registered, policy bool }{{false, false}, {true, false}, {true, true}} {
		registered := test.registered
		config := blackmagicConfig()
		if registered {
			config.Registration = &BlackmagicRegistration{Firstname: "Alex", Lastname: "Example", Email: "alex@example.test", Phone: "123456", Street: "123 Example Street", City: "Example City", Country: "au", Policy: test.policy}
		}
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodPost || req.URL.String() != "https://www.blackmagicdesign.com/api/register/us/download/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
				t.Fatalf("unexpected request %s %s", req.Method, req.URL)
			}
			var body map[string]any
			if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["platform"] != "Mac OS X" || body["product"] != "DaVinci Resolve" || req.Header.Get("Content-Type") != "application/json;charset=UTF-8" {
				t.Fatalf("registration=%v", body)
			}
			if body["policy"] != test.policy {
				t.Fatalf("unexpected registration consent: %v", body)
			}
			if registered && (body["street"] != "123 Example Street" || body["country"] != "au" || body["email"] != "alex@example.test" || body["phone"] != "123456" || body["city"] != "Example City" || body["firstname"] != "Alex" || body["lastname"] != "Example") {
				t.Fatalf("registration=%v", body)
			}
			if !registered && (len(body) != 4 || body["country"] != "us") {
				t.Fatalf("unexpected registration=%v", body)
			}
			return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("https://sw.blackmagicdesign.com/Resolve/Resolve_20.1_Mac.zip?token=private\n")), Header: make(http.Header), Request: req}, nil
		})}
		got, err := BlackmagicDownload(t.Context(), client, config, BlackmagicRelease{DownloadID: strings.Repeat("a", 32), Version: "20.1", RequiresRegistration: registered})
		if err != nil || got.Version != "20.1" || got.Filename != "Resolve_20.1_Mac.zip" {
			t.Fatalf("download=%+v, error=%v", got, err)
		}
	}
}

func TestBlackmagicRejectsDownloadResponsesWithoutLeakingBody(t *testing.T) {
	for _, test := range []struct {
		status int
		body   string
	}{
		{400, "PRIVATE registration details"}, {200, "PRIVATE invalid URL"},
		{200, "https://blackmagicdesign.com.evil.test/app.zip?PRIVATE"},
		{200, "http://sw.blackmagicdesign.com/app.zip?PRIVATE"},
		{200, "https://PRIVATE@sw.blackmagicdesign.com/app.zip"},
		{200, strings.Repeat("PRIVATE", 2000)},
		{307, "PRIVATE redirect"},
	} {
		client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: test.status, Body: io.NopCloser(strings.NewReader(test.body)), Header: http.Header{"Location": []string{"https://elsewhere.test/PRIVATE"}}, Request: req}, nil
		})}
		_, err := BlackmagicDownload(t.Context(), client, blackmagicConfig(), BlackmagicRelease{DownloadID: strings.Repeat("a", 32), Version: "21.1"})
		if err == nil || strings.Contains(err.Error(), "PRIVATE") {
			t.Fatalf("status %d: %v", test.status, err)
		}
	}
}

func TestBlackmagicRejectsMissingRegistrationBeforeHTTP(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected HTTP request")
		return nil, fmt.Errorf("unexpected HTTP")
	})}
	_, err := BlackmagicDownload(t.Context(), client, blackmagicConfig(), BlackmagicRelease{DownloadID: strings.Repeat("a", 32), Version: "21.1", RequiresRegistration: true})
	if err == nil || !strings.Contains(err.Error(), "requires registration") {
		t.Fatalf("missing registration: %v", err)
	}
}
