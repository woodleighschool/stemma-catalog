package downloads

import (
	"context"
	"encoding/json"
	"errors"
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
		plugin.Register(registry, resolver("blackmagic"), blackmagic(client)),
		plugin.Register(registry, resolver("python"), configured(client, "python", discovery.Python, nil)),
		plugin.Register(registry, resolver("cricut"), configured(client, "cricut", discovery.Cricut, discovery.CricutDownload)),
		plugin.Register(registry, resolver("microsoft"), configured(client, "microsoft", discovery.Microsoft, nil)),
		plugin.Register(registry, resolver("epson"), configured(client, "epson", discovery.Epson, nil)),
		plugin.Register(registry, resolver("unity"), configured(client, "unity", discovery.Unity, nil)),
	} {
		if err != nil {
			return err
		}
	}
	return nil
}

func resolver(name string) plugin.Operation {
	return plugin.Operation{Name: name, Kind: "resolve", Resolver: &plugin.ResolverKind{Version: "2"}, Methods: []string{"validate", "discover", "run"}, SideEffects: "none"}
}

func configured[C any](client *http.Client, name string, discover func(context.Context, *http.Client, C) (discovery.Release, error), downloadURL func(context.Context, *http.Client, C, discovery.Release) (string, error)) func(context.Context, plugin.ResolveRequest[C]) (plugin.ResolveResponse, error) {
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
			if err := validateRelease(release); err != nil {
				return plugin.ResolveResponse{}, err
			}
			if release.Signed {
				release.URL = ""
			}
			observation, err := json.Marshal(release)
			// A vendor version names one build, so it always downloads the same bytes.
			return plugin.ResolveResponse{Observation: observation, Version: release.Version, Immutable: release.Version != ""}, err
		}
		var release discovery.Release
		if err := json.Unmarshal(request.Observation, &release); err != nil {
			return plugin.ResolveResponse{}, errors.New("invalid download observation")
		}
		if downloadURL != nil {
			address, err := downloadURL(ctx, client, request.Config, release)
			if err != nil {
				return plugin.ResolveResponse{}, err
			}
			release.URL = address
		}
		if err := validateRelease(release); err != nil {
			return plugin.ResolveResponse{}, err
		}
		response := plugin.ResolveResponse{Download: &plugin.Download{URL: release.URL, Filename: release.Filename}}
		if release.Version != "" {
			version, err := json.Marshal(map[string]string{"version": release.Version})
			if err != nil {
				return plugin.ResolveResponse{}, err
			}
			response.Evidence = map[string]json.RawMessage{name: version}
		}
		return response, nil
	}
}
