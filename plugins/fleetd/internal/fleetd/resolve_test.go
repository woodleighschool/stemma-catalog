package fleetd

import (
	"crypto"
	"crypto/ed25519"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sigstore/sigstore/pkg/signature"
	"github.com/theupdateframework/go-tuf/v2/metadata"
	"github.com/woodleighschool/stemma/plugin"
)

func signedRepository(t *testing.T, expires time.Time, mismatchedAlias bool) ([]byte, map[string][]byte) {
	t.Helper()
	expiry := time.Now().Add(time.Hour)
	root := metadata.Root(expiry)
	root.Signed.ConsistentSnapshot = false
	targets := metadata.Targets(expiry)
	snapshot := metadata.Snapshot(expiry)
	timestamp := metadata.Timestamp(expires)
	body := []byte("synthetic orbit binary")
	hash := sha512.Sum512(body)
	custom := json.RawMessage(`{"version":"1.2.3"}`)
	for _, channel := range []string{"stable", "1.2.3"} {
		target := metadata.TargetFile()
		target.Length = int64(len(body))
		target.Hashes = metadata.Hashes{"sha512": hash[:]}
		target.Custom = &custom
		if mismatchedAlias && channel == "1.2.3" {
			target.Length++
		}
		targets.Signed.Targets["orbit/macos/"+channel+"/orbit"] = target
	}
	_, key, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	public, err := metadata.KeyFromPublicKey(key.Public())
	if err != nil {
		t.Fatal(err)
	}
	signer, err := signature.LoadSigner(key, crypto.Hash(0))
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"root", "targets", "snapshot", "timestamp"} {
		if err := root.Signed.AddKey(public, role); err != nil {
			t.Fatal(err)
		}
	}
	for _, sign := range []func(signature.Signer) (*metadata.Signature, error){root.Sign, targets.Sign, snapshot.Sign, timestamp.Sign} {
		if _, err := sign(signer); err != nil {
			t.Fatal(err)
		}
	}
	files := map[string][]byte{"/targets/orbit/macos/1.2.3/orbit": body}
	for name, document := range map[string]any{"root": root, "targets": targets, "snapshot": snapshot, "timestamp": timestamp} {
		data, err := json.Marshal(document)
		if err != nil {
			t.Fatal(err)
		}
		files["/"+name+".json"] = data
	}
	return files["/root.json"], files
}

func TestDiscoveryAuthenticatesMetadataAndPinsVersionedBytes(t *testing.T) {
	root, files := signedRepository(t, time.Now().Add(time.Hour), false)
	var frozen atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if frozen.Load() && request.URL.Path != "/targets/orbit/macos/1.2.3/orbit" {
			t.Error("frozen acquisition requested mutable metadata")
			http.NotFound(w, request)
			return
		}
		if body, ok := files[request.URL.Path]; ok {
			_, _ = w.Write(body)
		} else {
			http.NotFound(w, request)
		}
	}))
	t.Cleanup(server.Close)
	r := resolver{client: server.Client(), url: server.URL, root: root}
	s := selection{Component: "orbit", Channel: "stable"}
	found, err := r.resolve(t.Context(), plugin.ResolveRequest[selection]{Method: "discover", Config: s})
	if err != nil {
		t.Fatal(err)
	}
	if found.Version != "1.2.3" || !found.Immutable || found.Artifact.Path != "" {
		t.Fatalf("unexpected discovery: %+v", found)
	}
	frozen.Store(true)
	result, err := r.resolve(t.Context(), plugin.ResolveRequest[selection]{Method: "run", Config: s, Observation: found.Observation, Workspace: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(result.Artifact.Path)
	if err != nil || string(data) != "synthetic orbit binary" {
		t.Fatalf("acquired %q: %v", data, err)
	}
}

func TestDiscoveryRejectsUntrustedExpiredAndInconsistentMetadata(t *testing.T) {
	for _, kind := range []string{"signature", "expired", "alias"} {
		t.Run(kind, func(t *testing.T) {
			expiry := time.Now().Add(time.Hour)
			if kind == "expired" {
				expiry = time.Now().Add(-time.Hour)
			}
			root, files := signedRepository(t, expiry, kind == "alias")
			if kind == "signature" {
				_, untrusted := signedRepository(t, expiry, false)
				files["/targets.json"] = untrusted["/targets.json"]
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
				if body, ok := files[request.URL.Path]; ok {
					_, _ = w.Write(body)
				} else {
					http.NotFound(w, request)
				}
			}))
			t.Cleanup(server.Close)
			r := resolver{client: server.Client(), url: server.URL, root: root}
			if _, err := r.discover(t.Context(), selection{Component: "orbit", Channel: "stable"}); err == nil {
				t.Fatal("accepted invalid metadata")
			}
		})
	}
}

func TestDownloadRejectsChangedContentAndRemovesPartialFiles(t *testing.T) {
	for _, body := range []string{"corrupt!", "short", "longer than locked"} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(body)) }))
			t.Cleanup(server.Close)
			hash := sha512.Sum512([]byte("original"))
			found := observation{Version: "1", Target: "orbit/macos/1/orbit", Length: 8, SHA512: hex.EncodeToString(hash[:])}
			r := resolver{client: server.Client(), url: server.URL}
			workspace := t.TempDir()
			_, err := r.download(t.Context(), selection{Component: "orbit"}, found, workspace)
			if err == nil {
				t.Fatal("accepted changed content")
			}
			if _, err := os.Stat(filepath.Join(workspace, "orbit")); !os.IsNotExist(err) {
				t.Fatalf("partial download left behind: %v", err)
			}
		})
	}
}

func TestObservationRejectsPathsOutsideTheSelectedComponent(t *testing.T) {
	hash := sha512.Sum512([]byte("original"))
	for _, target := range []string{"../orbit", "desktop/macos/1/orbit", "orbit/macos/stable/orbit", "https://example.test/orbit"} {
		found := observation{Version: "1", Target: target, Length: 8, SHA512: hex.EncodeToString(hash[:])}
		if err := found.validate(selection{Component: "orbit"}); err == nil {
			t.Fatalf("accepted %q", target)
		}
	}
}
