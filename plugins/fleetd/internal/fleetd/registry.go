// Package fleetd resolves Fleet components and prepares their macOS installation layout.
package fleetd

import (
	"embed"
	"net/http"
	"time"

	"github.com/woodleighschool/stemma/plugin"
)

//go:embed assets/*
var assets embed.FS

const updateURL = "https://updates.fleetdm.com"

func Register(registry *plugin.Registry) error {
	root, err := assets.ReadFile("assets/root.json")
	if err != nil {
		return err
	}
	r := resolver{client: &http.Client{Timeout: 5 * time.Minute}, url: updateURL, root: root}
	if err := plugin.Register(registry, plugin.Operation{
		Name: "fleetd", Kind: "resolve", Resolver: &plugin.ResolverKind{Version: "1"},
		Methods: []string{"validate", "discover", "run"}, SideEffects: "workspace",
	}, r.resolve); err != nil {
		return err
	}
	return plugin.Register(registry, plugin.Operation{
		Name: "fleetd.payload", Kind: "resource",
		Resource: &plugin.ResourceKind{APIVersion: "fleetd/v1alpha1", Kind: "FleetMacPayload"},
		Methods:  []string{"discover", "run"}, SideEffects: "workspace",
	}, prepare)
}
