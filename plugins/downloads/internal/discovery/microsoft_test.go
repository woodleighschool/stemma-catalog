package discovery

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

const microsoftTestFeed = `<?xml version="1.0" encoding="UTF-8"?>
<plist version="1.0"><array>
<dict><key>Location</key><string>https://res.public.onecdn.static.microsoft/Delta.pkg</string>
<key>FullUpdaterLocation</key><string>https://res.public.onecdn.static.microsoft/Full.pkg</string>
<key>Update Version</key><string>16.114</string></dict>
<dict><key>Location</key><string>
 https://res.public.onecdn.static.microsoft/Microsoft_Outlook_16.113_Updater.pkg
</string><key>Update Version</key><string> 16.113 </string></dict>
<dict><key>Location</key><string>https://res.public.onecdn.static.microsoft/Microsoft_Outlook_16.100_Updater.pkg</string>
<key>Update Version</key><string>16.100</string></dict>
</array></plist>`

func TestMicrosoftSelectsFirstFullUpdate(t *testing.T) {
	for _, test := range []struct {
		name, channel, kind, channelID, suffix string
	}{
		{"defaults", "", "", "C1297A47-86C4-4C1F-97FA-950631F94777", "Installer"},
		{"explicit", "production", "standalone", "C1297A47-86C4-4C1F-97FA-950631F94777", "Installer"},
		{"preview", "preview", "", "1ac37578-5a24-40fb-892e-b89d85b6dfaa", "Installer"},
		{"beta updater", "beta", "updater", "4B2D7701-0A4F-49C8-B4CB-0C2D4043F51F", "Updater"},
	} {
		t.Run(test.name, func(t *testing.T) {
			endpoint := "https://res.public.onecdn.static.microsoft/mro1cdnstorage/" + test.channelID + "/MacAutoupdate/0409OPIM2019.xml"
			client := fixtureClient(t, map[string]string{endpoint: microsoftTestFeed})
			got, err := Microsoft(t.Context(), client, MicrosoftConfig{Product: "outlook", Channel: test.channel, Type: test.kind})
			filename := "Microsoft_Outlook_16.113_" + test.suffix + ".pkg"
			want := Release{URL: "https://res.public.onecdn.static.microsoft/" + filename, Filename: filename, Version: "16.113"}
			if err != nil || got != want {
				t.Fatalf("release = %+v, want %+v, error = %v", got, want, err)
			}
		})
	}
}

func TestMicrosoftOneNoteStandaloneUsesFullPackage(t *testing.T) {
	endpoint := "https://res.public.onecdn.static.microsoft/mro1cdnstorage/C1297A47-86C4-4C1F-97FA-950631F94777/MacAutoupdate/0409ONMC2019.xml"
	client := fixtureClient(t, map[string]string{endpoint: strings.ReplaceAll(microsoftTestFeed, "Outlook", "OneNote")})
	got, err := Microsoft(t.Context(), client, MicrosoftConfig{Product: "onenote"})
	want := Release{URL: "https://res.public.onecdn.static.microsoft/Microsoft_OneNote_16.113_Updater.pkg", Filename: "Microsoft_OneNote_16.113_Updater.pkg", Version: "16.113"}
	if err != nil || got != want {
		t.Fatalf("release = %+v, want %+v, error = %v", got, want, err)
	}
}

func TestMicrosoftRejectsInvalidManifest(t *testing.T) {
	for name, body := range map[string]string{
		"invalid plist":   "not a plist",
		"empty feed":      `<plist version="1.0"><array/></plist>`,
		"delta only":      `<plist version="1.0"><array><dict><key>FullUpdaterLocation</key><string>https://res.public.onecdn.static.microsoft/Full.pkg</string></dict></array></plist>`,
		"missing version": strings.Replace(microsoftTestFeed, " 16.113 ", "", 1),
		"insecure URL":    strings.ReplaceAll(microsoftTestFeed, "https://", "http://"),
		"foreign URL":     strings.ReplaceAll(microsoftTestFeed, "res.public.onecdn.static.microsoft", "microsoft.com.example.test"),
		"unknown suffix":  strings.ReplaceAll(microsoftTestFeed, "_Updater.pkg", "_Update.pkg"),
	} {
		t.Run(name, func(t *testing.T) {
			endpoint := "https://res.public.onecdn.static.microsoft/mro1cdnstorage/C1297A47-86C4-4C1F-97FA-950631F94777/MacAutoupdate/0409OPIM2019.xml"
			client := fixtureClient(t, map[string]string{endpoint: body})
			if _, err := Microsoft(t.Context(), client, MicrosoftConfig{Product: "outlook"}); err == nil {
				t.Fatal("accepted invalid manifest")
			}
		})
	}
}

func TestMicrosoftStandaloneResolvesWithoutDownloading(t *testing.T) {
	for product, link := range map[string]string{
		"office": "2009112", "defender": "2097502", "edge": "2093504",
		"teams": "2249065", "company-portal": "853070", "onedrive": "823060", "windows-app": "868963",
	} {
		t.Run(product, func(t *testing.T) {
			const target = "https://res.public.onecdn.static.microsoft/releases/Standalone.pkg"
			calls := 0
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				calls++
				if req.Method != http.MethodHead {
					t.Fatalf("discovery requested installer bytes: %s", req.Method)
				}
				response := &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: req}
				switch req.URL.String() {
				case "https://go.microsoft.com/fwlink/?linkid=" + link:
					response.StatusCode = http.StatusFound
					response.Header.Set("Location", target)
				case target:
				default:
					t.Fatalf("unexpected request %s", req.URL)
				}
				return response, nil
			})}
			got, err := Microsoft(t.Context(), client, MicrosoftConfig{Product: product})
			want := Release{URL: target, Filename: "Standalone.pkg"}
			if err != nil || got != want || calls != 2 {
				t.Fatalf("release = %+v, want %+v, calls = %d, error = %v", got, want, calls, err)
			}
		})
	}
}

func TestMicrosoftRejectsStandaloneRedirectsAndErrors(t *testing.T) {
	for _, target := range []string{"http://download.microsoft.com/App.pkg", "https://example.test/App.pkg", "https://download.microsoft.com/App.html"} {
		t.Run(target, func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				response := &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: req}
				if req.URL.Host == "go.microsoft.com" {
					response.StatusCode = http.StatusFound
					response.Header.Set("Location", target)
				}
				return response, nil
			})}
			if _, err := Microsoft(t.Context(), client, MicrosoftConfig{Product: "edge"}); err == nil {
				t.Fatal("accepted invalid standalone location")
			}
		})
	}
	for _, status := range []int{http.StatusNotFound, http.StatusMethodNotAllowed} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: status, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("")), Request: req}, nil
			})}
			if _, err := Microsoft(t.Context(), client, MicrosoftConfig{Product: "edge"}); err == nil {
				t.Fatal("accepted unsuccessful standalone request")
			}
		})
	}
}
