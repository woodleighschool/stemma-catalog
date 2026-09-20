package discovery

import (
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
)

// AudinateConfig selects a product's macOS release feed.
type AudinateConfig struct {
	Product string `json:"product" jsonschema:"enum=dante-controller,enum=dante-virtual-soundcard" jsonschema_description:"Dante product. Controller uses the Apple Silicon feed; Virtual Soundcard uses the macOS feed."`
}

// Audinate selects the first full macOS installer in the vendor's ordered feed.
func Audinate(ctx context.Context, client *http.Client, config AudinateConfig) (Release, error) {
	var feed string
	switch config.Product {
	case "dante-controller":
		feed = "/DanteController/appcast/DanteController-apple_silicon.xml"
	case "dante-virtual-soundcard":
		feed = "/DanteVirtualSoundcard/appcast/macOS/DanteVirtualSoundcard-macOS.xml"
	default:
		return Release{}, errors.New("audinate: unknown product")
	}
	endpoint := &url.URL{Scheme: "https", Host: "software-updates.audinate.com", Path: feed}
	data, err := metadata(ctx, client, endpoint, hostURL(endpoint.Host))
	if err != nil {
		return Release{}, fmt.Errorf("audinate: appcast: %w", err)
	}
	var appcast struct {
		XMLName xml.Name `xml:"rss"`
		Items   []struct {
			Version    string `xml:"http://www.andymatuschak.org/xml-namespaces/sparkle shortVersionString"`
			Enclosures []struct {
				URL     string `xml:"url,attr"`
				Version string `xml:"http://www.andymatuschak.org/xml-namespaces/sparkle shortVersionString,attr"`
				Build   string `xml:"http://www.andymatuschak.org/xml-namespaces/sparkle version,attr"`
				Delta   string `xml:"http://www.andymatuschak.org/xml-namespaces/sparkle deltaFrom,attr"`
			} `xml:"enclosure"`
		} `xml:"channel>item"`
	}
	if err := xml.Unmarshal(data, &appcast); err != nil {
		return Release{}, fmt.Errorf("audinate: appcast XML: %w", err)
	}
	for _, item := range appcast.Items {
		for _, enclosure := range item.Enclosures {
			if enclosure.Delta != "" {
				continue
			}
			u, err := url.Parse(enclosure.URL)
			if err != nil || !httpsURL(u) || !strings.HasSuffix(u.Path, ".dmg") || !validFilename(path.Base(u.Path)) {
				return Release{}, errors.New("audinate: enclosure must identify an HTTPS disk image")
			}
			version := enclosure.Version
			if version == "" {
				version = item.Version
			}
			if version == "" {
				version = enclosure.Build
			}
			if !validText(version) {
				return Release{}, errors.New("audinate: enclosure has no release version")
			}
			return Release{URL: u.String(), Filename: path.Base(u.Path), Version: version}, nil
		}
	}
	return Release{}, errors.New("audinate: appcast has no full installer")
}
