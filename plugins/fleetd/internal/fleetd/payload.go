package fleetd

import (
	"archive/tar"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/woodleighschool/stemma/plugin"
	"howett.net/plist"
)

type spec struct {
	FleetURL               string `json:"fleet_url,omitempty" jsonschema_description:"Fleet server HTTPS URL. Required with enroll_secret unless system configuration supplies enrollment."`
	EnrollSecret           string `json:"enroll_secret,omitempty" jsonschema:"writeOnly=true" jsonschema_description:"Enrollment secret embedded in the package. Supply through an environment expression; the built package contains this secret."`
	UseSystemConfiguration bool   `json:"use_system_configuration,omitempty" jsonschema_description:"Read enrollment settings from the endpoint's macOS configuration profile."`
	EnableScripts          bool   `json:"enable_scripts,omitempty" jsonschema_description:"Allow Fleet to execute scripts on the enrolled Mac."`
	DisableUpdates         bool   `json:"disable_updates,omitempty" jsonschema_description:"Disable agent self-updates so package deployment owns component versions."`
	OrbitChannel           string `json:"orbit_channel,omitempty" jsonschema:"default=stable" jsonschema_description:"Orbit update channel or component version to resolve."`
	OsquerydChannel        string `json:"osqueryd_channel,omitempty" jsonschema:"default=stable" jsonschema_description:"osquery update channel or component version to resolve."`
	Desktop                bool   `json:"desktop,omitempty" jsonschema_description:"Include and start Fleet Desktop."`
	DesktopChannel         string `json:"desktop_channel,omitempty" jsonschema:"default=stable" jsonschema_description:"Fleet Desktop update channel or component version to resolve when desktop is enabled."`
}

func (s spec) validate() error {
	for _, channel := range []string{s.OrbitChannel, s.OsquerydChannel, s.DesktopChannel} {
		if !channelName.MatchString(channel) {
			return errors.New("component channels must be Fleet channel or version names")
		}
	}
	if (s.FleetURL == "") != (s.EnrollSecret == "") || (!s.UseSystemConfiguration && s.FleetURL == "") {
		return errors.New("fleet_url and enroll_secret must be supplied together, or enable use_system_configuration")
	}
	if s.FleetURL != "" {
		u, err := url.Parse(s.FleetURL)
		if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
			return errors.New("fleet_url must be an HTTPS server URL without credentials, query or path")
		}
	}
	if strings.TrimSpace(s.EnrollSecret) != s.EnrollSecret || strings.ContainsAny(s.EnrollSecret, "\x00\r\n") {
		return errors.New("enroll_secret must be a single nonempty line without surrounding whitespace")
	}
	return nil
}

func prepare(ctx context.Context, request plugin.ResourceRequest[spec]) (plugin.ResourceResult, error) {
	s := request.Config
	if request.Method == "discover" {
		inputs := map[string]plugin.Input{}
		for component, channel := range map[string]string{"orbit": s.OrbitChannel, "osqueryd": s.OsquerydChannel, "desktop": s.DesktopChannel} {
			if component == "desktop" && !s.Desktop {
				continue
			}
			inputs[component] = plugin.Input{Resolver: "fleetd", Config: map[string]any{"component": component, "channel": channel}}
		}
		data, err := json.Marshal(s)
		return plugin.ResourceResult{Inputs: inputs, Config: data}, err
	}
	if err := s.validate(); err != nil {
		return plugin.ResourceResult{}, err
	}
	done := plugin.Stage(ctx, "Preparing Fleet installation layout")
	result, err := buildPayload(ctx, request)
	done(err)
	return result, err
}

func buildPayload(ctx context.Context, request plugin.ResourceRequest[spec]) (_ plugin.ResourceResult, err error) {
	s := request.Config
	filename := filepath.Join(request.Workspace, "payload.tar")
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return plugin.ResourceResult{}, err
	}
	defer func() { _ = f.Close() }()
	w := tar.NewWriter(f)
	defer func() { _ = w.Close() }()
	if err := w.WriteHeader(&tar.Header{Name: "var/log/orbit", Typeflag: tar.TypeDir, Mode: 0o755, Format: tar.FormatPAX}); err != nil {
		return plugin.ResourceResult{}, err
	}
	components := []string{"orbit", "osqueryd"}
	if s.Desktop {
		components = append(components, "desktop")
	}
	evidence := map[string]json.RawMessage{}
	for _, component := range components {
		input, ok := request.Inputs[component]
		if !ok || input.Tree || input.Path == "" {
			return plugin.ResourceResult{}, fmt.Errorf("missing %s component file", component)
		}
		evidence[component] = input.Evidence["fleetd"]
		channel := map[string]string{"orbit": s.OrbitChannel, "osqueryd": s.OsquerydChannel, "desktop": s.DesktopChannel}[component]
		selected := selection{Component: component, Channel: channel}
		platform, basename, _ := selected.target()
		directory := "opt/orbit/bin/" + component + "/" + platform + "/" + channel
		if component == "orbit" {
			if err := appendFile(ctx, w, directory+"/"+basename, input.Path, 0o755); err != nil {
				return plugin.ResourceResult{}, err
			}
		} else {
			bundle := "osquery.app"
			if component == "desktop" {
				bundle = "Fleet Desktop.app"
			}
			digest, err := appendBundle(ctx, w, directory, bundle, input.Path)
			if err != nil {
				return plugin.ResourceResult{}, fmt.Errorf("%s archive: %w", component, err)
			}
			if err := appendText(w, directory+"/"+basename+".sha512", digest, 0o600); err != nil {
				return plugin.ResourceResult{}, err
			}
		}
	}
	launchd, err := launchDaemon(s)
	if err != nil {
		return plugin.ResourceResult{}, err
	}
	certs, err := assets.ReadFile("assets/certs.pem")
	if err != nil {
		return plugin.ResourceResult{}, err
	}
	license, err := assets.ReadFile("assets/FLEET-LICENSE")
	if err != nil {
		return plugin.ResourceResult{}, err
	}
	for _, file := range []struct {
		name, contents string
		mode           int64
	}{
		{"Library/LaunchDaemons/com.fleetdm.orbit.plist", string(launchd), 0o644},
		{"opt/orbit/certs.pem", string(certs), 0o644},
		{"opt/orbit/LICENSE", string(license), 0o644},
		{"opt/orbit/osquery.flags", "", 0o600},
	} {
		if err := appendText(w, file.name, file.contents, file.mode); err != nil {
			return plugin.ResourceResult{}, err
		}
	}
	if s.EnrollSecret != "" {
		if err := appendText(w, "opt/orbit/secret.txt", s.EnrollSecret, 0o600); err != nil {
			return plugin.ResourceResult{}, err
		}
	}
	if err := w.Close(); err != nil {
		return plugin.ResourceResult{}, err
	}
	if err := f.Close(); err != nil {
		return plugin.ResourceResult{}, err
	}
	hook, err := assets.ReadFile("assets/postinstall")
	if err != nil {
		return plugin.ResourceResult{}, err
	}
	script := strings.ReplaceAll(string(hook), "@ORBIT_CHANNEL@", s.OrbitChannel)
	postinstall := filepath.Join(request.Workspace, "postinstall")
	if err := os.WriteFile(postinstall, []byte(script), 0o755); err != nil {
		return plugin.ResourceResult{}, err
	}
	data, err := json.Marshal(evidence)
	if err != nil {
		return plugin.ResourceResult{}, err
	}
	return plugin.ResourceResult{Artifacts: map[string]plugin.Artifact{
		"payload":     {Path: filename, Filename: "payload.tar", Format: "tar", Evidence: map[string]json.RawMessage{"fleetd": data}},
		"postinstall": {Path: postinstall, Filename: "postinstall"},
	}}, nil
}

func launchDaemon(s spec) ([]byte, error) {
	env := map[string]string{
		"ORBIT_ORBIT_CHANNEL":    s.OrbitChannel,
		"ORBIT_OSQUERYD_CHANNEL": s.OsquerydChannel,
		"ORBIT_UPDATE_URL":       updateURL,
		"ORBIT_UPDATE_INTERVAL":  "15m0s",
	}
	if s.FleetURL != "" {
		env["ORBIT_FLEET_URL"] = s.FleetURL
		env["ORBIT_ENROLL_SECRET_PATH"] = "/opt/orbit/secret.txt"
	}
	for name, enabled := range map[string]bool{"ORBIT_USE_SYSTEM_CONFIGURATION": s.UseSystemConfiguration,
		"ORBIT_ENABLE_SCRIPTS": s.EnableScripts, "ORBIT_DISABLE_UPDATES": s.DisableUpdates, "ORBIT_FLEET_DESKTOP": s.Desktop} {
		if enabled {
			env[name] = "true"
		}
	}
	if s.Desktop {
		env["ORBIT_DESKTOP_CHANNEL"] = s.DesktopChannel
	}
	return plist.MarshalIndent(map[string]any{
		"Label": "com.fleetdm.orbit", "ProgramArguments": []string{"/opt/orbit/bin/orbit/orbit"},
		"EnvironmentVariables": env, "KeepAlive": true, "RunAtLoad": true,
		"StandardErrorPath": "/var/log/orbit/orbit.stderr.log", "StandardOutPath": "/var/log/orbit/orbit.stdout.log",
		"ThrottleInterval": 10,
	}, plist.XMLFormat, "\t")
}

func appendText(w *tar.Writer, name, text string, mode int64) error {
	if err := w.WriteHeader(&tar.Header{Name: name, Typeflag: tar.TypeReg, Mode: mode, Size: int64(len(text)), Format: tar.FormatPAX}); err != nil {
		return err
	}
	_, err := io.WriteString(w, text)
	return err
}

func appendFile(ctx context.Context, w *tar.Writer, name, source string, mode int64) error {
	f, err := os.Open(source)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() > maxDownload {
		return errors.New("component must be a regular file within the download limit")
	}
	if err := w.WriteHeader(&tar.Header{Name: name, Mode: mode, Size: info.Size(), Typeflag: tar.TypeReg, Format: tar.FormatPAX}); err != nil {
		return err
	}
	_, err = io.Copy(w, contextReader{ctx, f})
	return err
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
