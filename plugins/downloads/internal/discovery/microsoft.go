// Feed IDs and Office updater selection adapted from AutoPkg's
// MSOfficeMacURLandUpdateInfoProvider.py, Copyright 2015 Allister Banks and
// Tim Sutton, licensed under the Apache License, Version 2.0.
// https://github.com/autopkg/recipes/blob/master/MSOfficeUpdates/MSOfficeMacURLandUpdateInfoProvider.py

package discovery

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/invopop/jsonschema"
	"howett.net/plist"
)

// MicrosoftConfig selects a macOS product, release channel and package type.
// Channel defaults to production and Type defaults to standalone.
type MicrosoftConfig struct {
	Product Product     `json:"product" jsonschema_description:"Microsoft macOS application. Office apps expose MAU channels; suite and standalone products support production installers only."`
	Channel Channel     `json:"channel,omitempty" jsonschema:"default=production" jsonschema_description:"Release feed. Preview and beta are available for individual Office apps only."`
	Type    PackageType `json:"type,omitempty" jsonschema:"default=standalone" jsonschema_description:"Standalone installer or full updater. Updaters are available for individual Office apps only."`
}

var microsoftApps = map[Product]string{
	"excel": "XCEL2019", "onenote": "ONMC2019", "outlook": "OPIM2019",
	"powerpoint": "PPT32019", "word": "MSWD2019",
}

var microsoftChannels = map[Channel]string{
	"production": "C1297A47-86C4-4C1F-97FA-950631F94777",
	"preview":    "1ac37578-5a24-40fb-892e-b89d85b6dfaa",
	"beta":       "4B2D7701-0A4F-49C8-B4CB-0C2D4043F51F",
}

// These products publish standalone installers separately from MAU updaters.
// Edge no longer publishes current releases through its MAU feed.
var microsoftInstallers = map[Product]string{
	"office": "2009112", "defender": "2097502", "edge": "2093504",
	"teams": "2249065", "company-portal": "853070", "onedrive": "823060",
	"windows-app": "868963",
}

// Microsoft resolves Office's MAU feed or a product's standalone download link.
func Microsoft(ctx context.Context, client *http.Client, config MicrosoftConfig) (Release, error) {
	if link, ok := microsoftInstallers[config.Product]; ok {
		return microsoftInstaller(ctx, client, link)
	}
	channel := microsoftChannels[config.Channel]
	app := microsoftApps[config.Product]
	endpoint := &url.URL{Scheme: "https", Host: "res.public.onecdn.static.microsoft", Path: "/mro1cdnstorage/" + channel + "/MacAutoupdate/0409" + app + ".xml"}
	data, err := metadata(ctx, client, endpoint, microsoftURL)
	if err != nil {
		return Release{}, fmt.Errorf("microsoft: manifest: %w", err)
	}
	var updates []struct {
		Location            string `plist:"Location"`
		FullUpdaterLocation string `plist:"FullUpdaterLocation"`
		Version             string `plist:"Update Version"`
	}
	if _, err := plist.Unmarshal(data, &updates); err != nil {
		return Release{}, fmt.Errorf("microsoft: manifest plist: %w", err)
	}
	// MAU orders current releases before releases retained for older macOS
	// versions. Delta entries require an existing version and are not selected.
	for _, update := range updates {
		if strings.TrimSpace(update.FullUpdaterLocation) != "" {
			continue
		}
		u, err := url.Parse(strings.TrimSpace(update.Location))
		if err != nil || !microsoftURL(u) {
			return Release{}, errors.New("microsoft: unexpected package URL")
		}
		// OneNote's official standalone link points to its full updater package.
		if config.Type == "standalone" && config.Product != "onenote" {
			base, ok := strings.CutSuffix(u.Path, "_Updater.pkg")
			if !ok {
				return Release{}, errors.New("microsoft: Office updater URL must end in _Updater.pkg")
			}
			u.Path, u.RawPath = base+"_Installer.pkg", ""
		}
		version := strings.TrimSpace(update.Version)
		if !validText(version) {
			return Release{}, errors.New("microsoft: manifest has no valid update version")
		}
		return microsoftPackage(u, version)
	}
	return Release{}, errors.New("microsoft: manifest has no full update")
}

func microsoftInstaller(ctx context.Context, client *http.Client, link string) (Release, error) {
	if client == nil {
		return Release{}, errors.New("microsoft: HTTP client is required")
	}
	bounded := *client
	bounded.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if !microsoftURL(req.URL) || len(via) >= 10 {
			return errors.New("microsoft: unexpected installer redirect")
		}
		if client.CheckRedirect != nil {
			return client.CheckRedirect(req, via)
		}
		return nil
	}
	// Resolve the public download link without reading installer bytes.
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, "https://go.microsoft.com/fwlink/?linkid="+link, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("User-Agent", "stemma-catalog-downloads")
	response, err := bounded.Do(req)
	if err != nil {
		return Release{}, fmt.Errorf("microsoft: standalone location: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("microsoft: standalone location returned HTTP %d", response.StatusCode)
	}
	return microsoftPackage(response.Request.URL, "")
}

func microsoftPackage(u *url.URL, version string) (Release, error) {
	filename := path.Base(u.Path)
	if !microsoftURL(u) || !validFilename(filename) || !strings.HasSuffix(filename, ".pkg") {
		return Release{}, errors.New("microsoft: expected a Microsoft PKG URL")
	}
	return Release{URL: u.String(), Filename: filename, Version: version}, nil
}

func microsoftURL(u *url.URL) bool {
	return httpsURL(u) && (strings.HasSuffix(u.Host, ".microsoft.com") ||
		strings.HasSuffix(u.Host, ".static.microsoft") || u.Host == "oneclient.sfx.ms" ||
		u.Host == "statics.teams.cdn.office.net")
}

// Validate enforces the release combinations actually published by Microsoft.
func (config MicrosoftConfig) Validate() error {
	if _, standalone := microsoftInstallers[config.Product]; standalone && (config.Channel != ChannelProduction || config.Type != PackageStandalone) {
		return fmt.Errorf("%s supports production standalone installers", config.Product)
	}
	return nil
}

// Product identifies a Microsoft macOS application.
type Product string

const (
	ProductExcel         Product = "excel"
	ProductOneNote       Product = "onenote"
	ProductOutlook       Product = "outlook"
	ProductPowerPoint    Product = "powerpoint"
	ProductWord          Product = "word"
	ProductOffice        Product = "office"
	ProductDefender      Product = "defender"
	ProductEdge          Product = "edge"
	ProductTeams         Product = "teams"
	ProductCompanyPortal Product = "company-portal"
	ProductOneDrive      Product = "onedrive"
	ProductWindowsApp    Product = "windows-app"
)

func (Product) JSONSchemaExtend(schema *jsonschema.Schema) {
	schema.Enum = []any{ProductExcel, ProductOneNote, ProductOutlook, ProductPowerPoint, ProductWord, ProductOffice, ProductDefender, ProductEdge, ProductTeams, ProductCompanyPortal, ProductOneDrive, ProductWindowsApp}
}

// Channel selects a Microsoft release feed.
type Channel string

const (
	ChannelProduction Channel = "production"
	ChannelPreview    Channel = "preview"
	ChannelBeta       Channel = "beta"
)

func (Channel) JSONSchemaExtend(schema *jsonschema.Schema) {
	schema.Enum = []any{ChannelProduction, ChannelPreview, ChannelBeta}
}

// PackageType selects a standalone installer or a full updater.
type PackageType string

const (
	PackageStandalone PackageType = "standalone"
	PackageUpdater    PackageType = "updater"
)

func (PackageType) JSONSchemaExtend(schema *jsonschema.Schema) {
	schema.Enum = []any{PackageStandalone, PackageUpdater}
}
