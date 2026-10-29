package fleetd

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"time"

	"github.com/theupdateframework/go-tuf/v2/metadata"
	"github.com/theupdateframework/go-tuf/v2/metadata/config"
	"github.com/theupdateframework/go-tuf/v2/metadata/updater"
	"github.com/woodleighschool/stemma/plugin"
)

const maxDownload = 1 << 30

var channelName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)

type selection struct {
	Component string `json:"component" jsonschema:"enum=orbit,enum=osqueryd,enum=desktop" jsonschema_description:"Fleet component to acquire for macOS."`
	Channel   string `json:"channel,omitempty" jsonschema:"default=stable" jsonschema_description:"Fleet update channel or published component version."`
}

func (s selection) Validate() error {
	if _, _, err := s.target(); err != nil {
		return err
	}
	if !channelName.MatchString(s.Channel) {
		return errors.New("channel must be a Fleet channel or version without path separators")
	}
	return nil
}

func (s selection) target() (platform, filename string, err error) {
	switch s.Component {
	case "orbit":
		return "macos", "orbit", nil
	case "osqueryd":
		return "macos-app", "osqueryd.app.tar.gz", nil
	case "desktop":
		return "macos", "desktop.app.tar.gz", nil
	default:
		return "", "", errors.New("component must be orbit, osqueryd or desktop")
	}
}

func (s selection) targetPath(version string) string {
	platform, filename, _ := s.target()
	return path.Join(s.Component, platform, version, filename)
}

type observation struct {
	Version string `json:"version"`
	Target  string `json:"target"`
	Length  int64  `json:"length"`
	SHA512  string `json:"sha512"`
}

func (o observation) validate(s selection) error {
	digest, err := hex.DecodeString(o.SHA512)
	if err != nil || len(digest) != sha512.Size || !channelName.MatchString(o.Version) ||
		o.Target != s.targetPath(o.Version) || o.Length <= 0 || o.Length > maxDownload {
		return errors.New("invalid fleetd target observation")
	}
	return nil
}

type resolver struct {
	client *http.Client
	url    string
	root   []byte
}

func (r resolver) resolve(ctx context.Context, request plugin.ResolveRequest[selection]) (plugin.ResolveResponse, error) {
	if request.Method == "discover" {
		done := plugin.Stage(ctx, "Resolving Fleet component", plugin.Detail(request.Config.Component))
		found, err := r.discover(ctx, request.Config)
		done(err, plugin.Detail(found.Version))
		if err != nil {
			return plugin.ResolveResponse{}, err
		}
		data, err := json.Marshal(found)
		return plugin.ResolveResponse{Observation: data, Version: found.Version, Immutable: true,
			Evidence: map[string]json.RawMessage{"fleetd": data}}, err
	}
	var found observation
	if err := json.Unmarshal(request.Observation, &found); err != nil {
		return plugin.ResolveResponse{}, errors.New("invalid fleetd target observation")
	}
	if err := found.validate(request.Config); err != nil {
		return plugin.ResolveResponse{}, err
	}
	artifact, err := r.download(ctx, request.Config, found, request.Workspace)
	return plugin.ResolveResponse{Artifact: artifact}, err
}

func (r resolver) discover(ctx context.Context, s selection) (observation, error) {
	cfg, err := config.New(r.url, r.root)
	if err != nil {
		return observation{}, fmt.Errorf("fleetd: TUF configuration: %w", err)
	}
	cfg.DisableLocalCache = true
	cfg.Fetcher = metadataFetcher{ctx: ctx, client: r.client}
	u, err := updater.New(cfg)
	if err != nil {
		return observation{}, fmt.Errorf("fleetd: TUF root: %w", err)
	}
	if err := u.Refresh(); err != nil {
		return observation{}, fmt.Errorf("fleetd: TUF refresh: %w", err)
	}
	target, err := u.GetTargetInfo(s.targetPath(s.Channel))
	if err != nil {
		return observation{}, fmt.Errorf("fleetd: channel: %w", err)
	}
	var custom struct {
		Version string `json:"version"`
	}
	if target.Custom == nil || json.Unmarshal(*target.Custom, &custom) != nil || !channelName.MatchString(custom.Version) {
		return observation{}, errors.New("fleetd: target has no valid component version")
	}
	found := observation{Version: custom.Version, Target: s.targetPath(custom.Version), Length: target.Length,
		SHA512: hex.EncodeToString(target.Hashes["sha512"])}
	if err := found.validate(s); err != nil {
		return observation{}, err
	}
	// Channel paths move. Only lock the versioned alias when its signed digest agrees.
	pinned, err := u.GetTargetInfo(found.Target)
	if err != nil {
		return observation{}, fmt.Errorf("fleetd: versioned target: %w", err)
	}
	if pinned.Length != found.Length || hex.EncodeToString(pinned.Hashes["sha512"]) != found.SHA512 {
		return observation{}, errors.New("fleetd: channel and versioned target disagree")
	}
	return found, nil
}

func (r resolver) download(ctx context.Context, s selection, found observation, workspace string) (_ plugin.Artifact, err error) {
	done := plugin.Stage(ctx, "Downloading Fleet component", plugin.Detail(s.Component+" "+found.Version))
	defer func() { done(err) }()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.url+"/targets/"+found.Target, nil)
	if err != nil {
		return plugin.Artifact{}, err
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return plugin.Artifact{}, fmt.Errorf("fleetd: download: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return plugin.Artifact{}, fmt.Errorf("fleetd: download returned HTTP %d", resp.StatusCode)
	}
	_, filename, _ := s.target()
	output := filepath.Join(workspace, filename)
	f, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return plugin.Artifact{}, err
	}
	defer func() {
		_ = f.Close()
		if err != nil {
			_ = os.Remove(output)
		}
	}()
	hash := sha512.New()
	n, err := io.Copy(io.MultiWriter(f, hash), plugin.ProgressReader(ctx, io.LimitReader(resp.Body, found.Length+1), found.Length))
	if err != nil {
		return plugin.Artifact{}, fmt.Errorf("fleetd: download: %w", err)
	}
	if n != found.Length || hex.EncodeToString(hash.Sum(nil)) != found.SHA512 {
		return plugin.Artifact{}, errors.New("fleetd: download does not match locked length and SHA-512")
	}
	if err := f.Close(); err != nil {
		return plugin.Artifact{}, err
	}
	return plugin.Artifact{Path: output, Filename: filename, Size: n}, nil
}

// The TUF fetcher has no context parameter; each discovery binds its own context.
type metadataFetcher struct {
	ctx    context.Context
	client *http.Client
}

func (f metadataFetcher) DownloadFile(address string, limit int64, _ time.Duration) ([]byte, error) {
	req, err := http.NewRequestWithContext(f.ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, &metadata.ErrDownloadHTTP{StatusCode: resp.StatusCode, URL: address}
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, errors.New("fleetd: metadata exceeds size limit")
	}
	return data, nil
}
