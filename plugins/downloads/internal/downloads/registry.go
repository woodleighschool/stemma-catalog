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
	return plugin.Operation{Name: name, Kind: "resolve", Resolver: &plugin.ResolverKind{Version: "1"}, Methods: []string{"validate", "run"}, SideEffects: "workspace"}
}

func configured[C any](client *http.Client, name string, discover func(context.Context, *http.Client, C) (discovery.Release, error)) func(context.Context, plugin.ResolveRequest[C]) (plugin.ResolveResponse, error) {
	return func(ctx context.Context, request plugin.ResolveRequest[C]) (plugin.ResolveResponse, error) {
		var release discovery.Release
		if request.Locked {
			if err := json.Unmarshal(request.Observation, &release); err != nil {
				return plugin.ResolveResponse{}, errors.New("invalid locked download observation")
			}
		} else {
			done := plugin.Stage(ctx, "Discovering vendor release")
			var err error
			release, err = discover(ctx, client, request.Config)
			done(err, plugin.Detail(release.Version))
			if err != nil {
				return plugin.ResolveResponse{}, err
			}
		}
		done := plugin.Stage(ctx, "Downloading release", plugin.Detail(release.Filename))
		artifact, err := download(ctx, client, release, request.Workspace)
		done(err)
		if err != nil {
			return plugin.ResolveResponse{}, err
		}
		observation, err := json.Marshal(release)
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
		return plugin.ResolveResponse{Observation: observation, Artifact: artifact}, nil
	}
}
