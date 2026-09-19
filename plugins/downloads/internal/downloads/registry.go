package downloads

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/invopop/jsonschema"
	"github.com/woodleighschool/stemma-catalog/plugins/downloads/internal/discovery"
	"github.com/woodleighschool/stemma/plugin"
)

// Register adds the catalog's vendor resolvers with stable operation names.
func Register(registry *plugin.Registry, client *http.Client) error {
	if client == nil {
		return errors.New("download resolvers require an HTTP client")
	}
	operations := []struct {
		name     string
		schema   string
		discover func(context.Context, *http.Client, json.RawMessage) (discovery.Release, error)
	}{
		{"blender", `{"type":"object","additionalProperties":false,"required":["major"],"properties":{"major":{"type":"integer","minimum":1},"architecture":{"type":"string","enum":["arm64","x64"],"default":"arm64"}}}`, configured(discovery.Blender)},
		{"python", `{"type":"object","additionalProperties":false,"required":["branch"],"properties":{"branch":{"type":"string","pattern":"^3\\.[1-9][0-9]+$"}}}`, configured(discovery.Python)},
		{"cricut", `{"type":"object","additionalProperties":false,"properties":{"operating_system":{"type":"string","enum":["osxnative"],"default":"osxnative"},"shard":{"type":"string","pattern":"^[A-Za-z0-9_-]+$","default":"a"}}}`, configured(discovery.Cricut)},
		{"microsoft", `{"type":"object","additionalProperties":false,"required":["product"],"properties":{"product":{"type":"string","enum":["excel","onenote","outlook","powerpoint","word","office","defender","edge","teams","company-portal","onedrive","windows-app"]},"channel":{"type":"string","enum":["production","preview","beta"],"default":"production"},"type":{"type":"string","enum":["standalone","updater"],"default":"standalone"}},"allOf":[{"if":{"properties":{"product":{"enum":["office","defender","edge","teams","company-portal","onedrive","windows-app"]}}},"then":{"properties":{"channel":{"const":"production"},"type":{"const":"standalone"}}}}]}`, configured(discovery.Microsoft)},
		{"epson", `{"type":"object","additionalProperties":false,"required":["device_id","os","cti"],"properties":{"device_id":{"type":"string","minLength":1},"os":{"type":"string","pattern":"^[A-Za-z0-9_-]+$"},"cti":{"type":"string","pattern":"^[0-9]+$"},"region":{"type":"string","pattern":"^[A-Z]{2}$","default":"GB"},"language":{"type":"string","pattern":"^[a-z]{2}(-[A-Z]{2})?$","default":"en"}}}`, configured(discovery.Epson)},
	}
	reflector := jsonschema.Reflector{DoNotReference: true, RequiredFromJSONSchemaTags: true}
	input := reflector.Reflect(plugin.ResolveRequest{})
	input.ID = ""
	inputSchema, err := json.Marshal(input)
	if err != nil {
		return err
	}
	output := reflector.Reflect(plugin.ResolveResponse{})
	output.ID = ""
	outputSchema, err := json.Marshal(output)
	if err != nil {
		return err
	}
	for _, operation := range operations {
		err := registry.Register(plugin.Operation{
			Name: operation.name, Kind: "resolve", Resolver: &plugin.ResolverKind{Version: "1"},
			Methods: []string{"validate", "run"}, SideEffects: "workspace",
			ConfigSchema: json.RawMessage(operation.schema), InputSchema: inputSchema, OutputSchema: outputSchema,
		}, func(ctx context.Context, envelope plugin.Request) (plugin.Response, error) {
			if envelope.Method == "validate" {
				return plugin.Response{}, nil
			}
			var request plugin.ResolveRequest
			if err := json.Unmarshal(envelope.Input, &request); err != nil {
				return plugin.Response{}, err
			}
			var release discovery.Release
			if request.Locked {
				if err := json.Unmarshal(request.Observation, &release); err != nil {
					return plugin.Response{}, errors.New("invalid locked download observation")
				}
			} else {
				done := plugin.Stage(ctx, "Discovering vendor release")
				var err error
				release, err = operation.discover(ctx, client, request.Config)
				done(err)
				if err != nil {
					return plugin.Response{}, fmt.Errorf("%s: %w", operation.name, err)
				}
			}
			done := plugin.Stage(ctx, "Downloading release")
			artifact, err := download(ctx, client, release, request.Workspace)
			done(err)
			if err != nil {
				return plugin.Response{}, err
			}
			observation, err := json.Marshal(release)
			if err != nil {
				return plugin.Response{}, err
			}
			if release.Version != "" {
				version, err := json.Marshal(map[string]string{"version": release.Version})
				if err != nil {
					return plugin.Response{}, err
				}
				artifact.Evidence = map[string]json.RawMessage{operation.name: version}
			}
			data, err := json.Marshal(plugin.ResolveResponse{Observation: observation, Artifact: artifact})
			return plugin.Response{Output: data}, err
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func configured[C any](discover func(context.Context, *http.Client, C) (discovery.Release, error)) func(context.Context, *http.Client, json.RawMessage) (discovery.Release, error) {
	return func(ctx context.Context, client *http.Client, data json.RawMessage) (discovery.Release, error) {
		var config C
		if err := json.Unmarshal(data, &config); err != nil {
			return discovery.Release{}, err
		}
		return discover(ctx, client, config)
	}
}
