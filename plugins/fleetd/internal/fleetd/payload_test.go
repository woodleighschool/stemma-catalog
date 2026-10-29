package fleetd

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/woodleighschool/stemma/plugin"
	"howett.net/plist"
)

type tarEntry struct {
	header tar.Header
	body   string
}

func writeBundle(t *testing.T, entries []tarEntry) string {
	t.Helper()
	var data bytes.Buffer
	gz := gzip.NewWriter(&data)
	w := tar.NewWriter(gz)
	for _, entry := range entries {
		h := entry.header
		if h.Typeflag == tar.TypeReg {
			h.Size = int64(len(entry.body))
		}
		if err := w.WriteHeader(&h); err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(w, entry.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "app.tar.gz")
	if err := os.WriteFile(file, data.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return file
}

func bundleEntries(bundle, executable string) []tarEntry {
	return []tarEntry{
		{tar.Header{Name: bundle + "/Contents/MacOS/agent", Mode: 0o755, Typeflag: tar.TypeReg}, executable},
		{tar.Header{Name: bundle + "/Contents/Info.plist", Mode: 0o644, Typeflag: tar.TypeReg}, "signed plist bytes"},
		{tar.Header{Name: bundle + "/Contents/_CodeSignature/CodeResources", Mode: 0o644, Typeflag: tar.TypeReg}, "signature bytes"},
		{tar.Header{Name: bundle + "/Contents/Resources/current", Mode: 0o755, Typeflag: tar.TypeSymlink, Linkname: "../Info.plist"}, ""},
	}
}

func readPayload(t *testing.T, name string) (map[string]tarEntry, []byte) {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	r := tar.NewReader(bytes.NewReader(data))
	entries := map[string]tarEntry{}
	for {
		h, err := r.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(r)
		if err != nil {
			t.Fatal(err)
		}
		entries[h.Name] = tarEntry{*h, string(body)}
	}
	return entries, data
}

func TestPayloadPreservesFleetLayoutWithoutExecutingInputs(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "executed")
	executable := "#!/bin/sh\ntouch '" + marker + "'\n"
	orbit := filepath.Join(t.TempDir(), "orbit")
	if err := os.WriteFile(orbit, []byte(executable), 0o755); err != nil {
		t.Fatal(err)
	}
	osquery := writeBundle(t, bundleEntries("osquery.app", executable))
	desktop := writeBundle(t, bundleEntries("Fleet Desktop.app", executable))
	s := spec{FleetURL: "https://fleet.example.test", EnrollSecret: "synthetic-enrollment", EnableScripts: true, DisableUpdates: true,
		OrbitChannel: "stable", OsquerydChannel: "stable", DesktopChannel: "stable", Desktop: true}
	inputs := map[string]plugin.Artifact{"orbit": {Path: orbit}, "osqueryd": {Path: osquery}, "desktop": {Path: desktop}}
	build := func() plugin.ResourceResult {
		t.Helper()
		result, err := prepare(t.Context(), plugin.ResourceRequest[spec]{Method: "run", Config: s, Inputs: inputs, Workspace: t.TempDir()})
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	result := build()
	entries, data := readPayload(t, result.Artifacts["payload"].Path)
	_, repeated := readPayload(t, build().Artifacts["payload"].Path)
	if !bytes.Equal(data, repeated) {
		t.Fatal("identical inputs produced different payloads")
	}
	for name, mode := range map[string]int64{
		"opt/orbit/secret.txt": 0o600, "opt/orbit/osquery.flags": 0o600, "opt/orbit/certs.pem": 0o644,
		"opt/orbit/bin/orbit/macos/stable/orbit": 0o755, "Library/LaunchDaemons/com.fleetdm.orbit.plist": 0o644, "var/log/orbit": 0o755,
	} {
		if entry, ok := entries[name]; !ok || entry.header.Mode != mode {
			t.Errorf("missing file or incorrect mode for %s", name)
		}
	}
	if entries["opt/orbit/secret.txt"].body != s.EnrollSecret {
		t.Error("enrollment secret was not preserved")
	}
	for _, name := range []string{"osquery.db", "identifier", "node.key", "tuf-metadata.json", "updates-metadata.json"} {
		if _, exists := entries["opt/orbit/"+name]; exists {
			t.Errorf("package replaces runtime state %s", name)
		}
	}
	for _, bundle := range []string{"osqueryd/macos-app/stable/osquery.app", "desktop/macos/stable/Fleet Desktop.app"} {
		prefix := "opt/orbit/bin/" + bundle + "/Contents/"
		if entries[prefix+"_CodeSignature/CodeResources"].body != "signature bytes" || entries[prefix+"Resources/current"].header.Linkname != "../Info.plist" {
			t.Error("bundle signature bytes or links changed")
		}
	}
	source, err := os.ReadFile(osquery)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha512.Sum512(source)
	if entries["opt/orbit/bin/osqueryd/macos-app/stable/osqueryd.app.tar.gz.sha512"].body != hex.EncodeToString(hash[:]) {
		t.Error("wrong installed archive checksum")
	}
	var daemon struct {
		Label                string
		ProgramArguments     []string
		EnvironmentVariables map[string]string
	}
	if _, err := plist.Unmarshal([]byte(entries["Library/LaunchDaemons/com.fleetdm.orbit.plist"].body), &daemon); err != nil {
		t.Fatal(err)
	}
	if daemon.Label != "com.fleetdm.orbit" || len(daemon.ProgramArguments) != 1 || daemon.ProgramArguments[0] != "/opt/orbit/bin/orbit/orbit" {
		t.Fatalf("unexpected daemon: %+v", daemon)
	}
	for key, want := range map[string]string{"ORBIT_FLEET_URL": s.FleetURL, "ORBIT_ENROLL_SECRET_PATH": "/opt/orbit/secret.txt", "ORBIT_DISABLE_UPDATES": "true", "ORBIT_ENABLE_SCRIPTS": "true", "ORBIT_FLEET_DESKTOP": "true"} {
		if daemon.EnvironmentVariables[key] != want {
			t.Errorf("incorrect %s", key)
		}
	}
	response, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(response, []byte(s.EnrollSecret)) {
		t.Error("enrollment secret leaked into artifact metadata")
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("payload was executed: %v", err)
	}
	hook, err := os.ReadFile(result.Artifacts["postinstall"].Path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(hook), "@ORBIT_CHANNEL@") || !strings.Contains(string(hook), "/macos/stable/orbit") {
		t.Error("postinstall does not select the installed Orbit")
	}
}

func TestConfigurationProfileOmitsPackagedEnrollment(t *testing.T) {
	s := spec{UseSystemConfiguration: true, OrbitChannel: "stable", OsquerydChannel: "stable", DesktopChannel: "stable"}
	if err := s.validate(); err != nil {
		t.Fatal(err)
	}
	data, err := launchDaemon(s)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("ORBIT_USE_SYSTEM_CONFIGURATION")) || bytes.Contains(data, []byte("ORBIT_ENROLL_SECRET_PATH")) {
		t.Fatal("incorrect profile enrollment settings")
	}
}

func TestBundleRejectsUnsafeOrUnsupportedMetadata(t *testing.T) {
	for _, h := range []tar.Header{
		{Name: "../escape", Typeflag: tar.TypeReg, Mode: 0o644},
		{Name: "osquery.app/../escape", Typeflag: tar.TypeReg, Mode: 0o644},
		{Name: "osquery.app/Contents/link", Typeflag: tar.TypeSymlink, Linkname: "../../../escape", Mode: 0o755},
		{Name: "osquery.app/Contents/link", Typeflag: tar.TypeLink, Linkname: "osquery.app/Contents/Info.plist", Mode: 0o755},
		{Name: "osquery.app/Contents/suid", Typeflag: tar.TypeReg, Mode: 0o4755},
		{Name: "osquery.app/Contents/attrs", Typeflag: tar.TypeReg, Mode: 0o644, PAXRecords: map[string]string{"SCHILY.xattr.security.test": "value"}},
		{Name: "osquery.app/Contents/Info.plist", Typeflag: tar.TypeReg, Mode: 0o644},
	} {
		t.Run(h.Name, func(t *testing.T) {
			entries := append(bundleEntries("osquery.app", "binary"), tarEntry{header: h})
			w := tar.NewWriter(io.Discard)
			_, err := appendBundle(t.Context(), w, "opt/orbit", "osquery.app", writeBundle(t, entries))
			_ = w.Close()
			if err == nil {
				t.Fatal("accepted unsupported archive")
			}
		})
	}
}

func TestResourceDiscoveryUsesDefaultsWithoutReadingEnrollment(t *testing.T) {
	registry := plugin.New("test", "1")
	if err := Register(registry); err != nil {
		t.Fatal(err)
	}
	response, err := registry.Handle(t.Context(), plugin.Request{Operation: "fleetd.payload", Method: "discover",
		Input: json.RawMessage(`{"identity":{"apiVersion":"fleetd/v1alpha1","kind":"FleetMacPayload","name":"test"},"config":{"fleet_url":"{{ env.FLEET_URL }}","enroll_secret":"{{ env.FLEET_ENROLL_SECRET }}"}}`)})
	if err != nil {
		t.Fatal(err)
	}
	var result plugin.ResourceResult
	if err := json.Unmarshal(response.Output, &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Inputs) != 2 || result.Inputs["orbit"].Config["channel"] != "stable" || result.Inputs["osqueryd"].Config["component"] != "osqueryd" {
		t.Fatalf("unexpected inputs: %+v", result.Inputs)
	}
}
