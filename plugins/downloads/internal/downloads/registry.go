package downloads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/woodleighschool/stemma-catalog/plugins/downloads/internal/discovery"
	"github.com/woodleighschool/stemma/plugin"
)

// Register adds the catalog's vendor resolvers.
func Register(registry *plugin.Registry, client *http.Client) error {
	if client == nil {
		return errors.New("download resolvers require an HTTP client")
	}
	for _, err := range []error{
		plugin.Register(registry, resolver("adobe"), configured(client, "adobe", discovery.Adobe)),
		plugin.Register(registry, resolver("audinate"), configured(client, "audinate", discovery.Audinate)),
		plugin.Register(registry, resolver("blender"), configured(client, "blender", discovery.Blender)),
		plugin.Register(registry, resolver("python"), configured(client, "python", discovery.Python)),
		plugin.Register(registry, resolver("cricut"), configured(client, "cricut", discovery.Cricut)),
		plugin.Register(registry, resolver("microsoft"), configured(client, "microsoft", discovery.Microsoft)),
		plugin.Register(registry, resolver("epson"), configured(client, "epson", discovery.Epson)),
	} {
		if err != nil {
			return err
		}
	}
	return nil
}

func resolver(name string) plugin.Operation {
	return plugin.Operation{Name: name, Kind: "resolve", Resolver: &plugin.ResolverKind{Version: "1"}, Methods: []string{"validate", "discover", "run"}, SideEffects: "workspace"}
}

func configured[C any](client *http.Client, name string, discover func(context.Context, *http.Client, C) (discovery.Release, error)) func(context.Context, plugin.ResolveRequest[C]) (plugin.ResolveResponse, error) {
	find := func(ctx context.Context, config C) (discovery.Release, error) {
		done := plugin.Stage(ctx, "Discovering vendor release")
		release, err := discover(ctx, client, config)
		done(err, plugin.Detail(release.Version))
		return release, err
	}
	return func(ctx context.Context, request plugin.ResolveRequest[C]) (plugin.ResolveResponse, error) {
		if request.Method == "discover" {
			release, err := find(ctx, request.Config)
			if err != nil {
				return plugin.ResolveResponse{}, err
			}
			if release.Signed {
				release.URL = ""
			}
			observation, err := json.Marshal(release)
			// A vendor version names one build, so it always downloads the same bytes.
			return plugin.ResolveResponse{Observation: observation, Immutable: release.Version != ""}, err
		}
		var release discovery.Release
		if err := json.Unmarshal(request.Observation, &release); err != nil {
			return plugin.ResolveResponse{}, errors.New("invalid download observation")
		}
		if release.URL == "" {
			current, err := find(ctx, request.Config)
			if err != nil {
				return plugin.ResolveResponse{}, err
			}
			if current.Filename != release.Filename || current.Version != release.Version {
				return plugin.ResolveResponse{}, fmt.Errorf("vendor no longer offers %s", release.Filename)
			}
			release.URL = current.URL
		}
		done := plugin.Stage(ctx, "Downloading release", plugin.Detail(release.Filename))
		artifact, err := download(ctx, client, release, request.Workspace)
		done(err)
		if err != nil {
			return plugin.ResolveResponse{}, err
		}
		if release.Version != "" {
			version, err := json.Marshal(map[string]string{"version": release.Version})
			if err != nil {
				return plugin.ResolveResponse{}, err
			}
			artifact.Evidence = map[string]json.RawMessage{name: version}
		}
		return plugin.ResolveResponse{Artifact: artifact}, nil
	}
}
