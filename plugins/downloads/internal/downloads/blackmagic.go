package downloads

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/woodleighschool/stemma-catalog/plugins/downloads/internal/discovery"
	"github.com/woodleighschool/stemma/plugin"
)

func blackmagic(client *http.Client) func(context.Context, plugin.ResolveRequest[discovery.BlackmagicConfig]) (plugin.ResolveResponse, error) {
	return func(ctx context.Context, request plugin.ResolveRequest[discovery.BlackmagicConfig]) (plugin.ResolveResponse, error) {
		if request.Method == "discover" {
			done := plugin.Stage(ctx, "Discovering vendor release")
			release, err := discovery.Blackmagic(ctx, client, request.Config)
			done(err, plugin.Detail(release.Version))
			if err != nil {
				return plugin.ResolveResponse{}, err
			}
			observation, err := json.Marshal(release)
			return plugin.ResolveResponse{Observation: observation, Version: release.Version, Immutable: true}, err
		}
		var observed discovery.BlackmagicRelease
		if err := json.Unmarshal(request.Observation, &observed); err != nil {
			return plugin.ResolveResponse{}, errors.New("blackmagic: invalid installer observation")
		}
		done := plugin.Stage(ctx, "Requesting Blackmagic download")
		release, err := discovery.BlackmagicDownload(ctx, client, request.Config, observed)
		done(err)
		if err != nil {
			return plugin.ResolveResponse{}, err
		}
		if err := validateRelease(release); err != nil {
			return plugin.ResolveResponse{}, err
		}
		version, err := json.Marshal(map[string]string{"version": release.Version})
		if err != nil {
			return plugin.ResolveResponse{}, err
		}
		return plugin.ResolveResponse{Download: &plugin.Download{URL: release.URL, Filename: release.Filename}, Evidence: map[string]json.RawMessage{"blackmagic": version}}, nil
	}
}
