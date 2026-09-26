package discovery

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

// AdobeConfig has no settings: the resolver selects the Creative Cloud desktop
// app installer for Apple silicon.
type AdobeConfig struct{}

// adobeInstaller matches an Apple silicon installer path. Its release and build
// directories together form the version the installer's ApplicationInfo.xml
// states, and the filename repeats it. Components are at most nine digits.
var adobeInstaller = regexp.MustCompile(`^/AdobeProducts/StandaloneBuilds/ACCC/ESD/([0-9]{1,9}(?:\.[0-9]{1,9})*)/([0-9]{1,9}(?:\.[0-9]{1,9})*)/macarm64/(ACCCx[0-9_]+\.dmg)$`)

// Adobe selects the newest Creative Cloud desktop installer for Apple silicon
// linked from Adobe's direct download page, which also links older builds.
func Adobe(ctx context.Context, client *http.Client, _ AdobeConfig) (Release, error) {
	page := &url.URL{Scheme: "https", Host: "helpx.adobe.com", Path: "/download-install/apps/download-install-apps/creative-cloud-apps/download-creative-cloud-desktop-app-using-direct-links.html"}
	data, err := metadataAs(ctx, client, page, hostURL(page.Host), "")
	if err != nil {
		return Release{}, fmt.Errorf("adobe: %w", err)
	}
	download := hostURL("ccmdls.adobe.com")
	var release Release
	var newest []int
	for _, value := range attributeValues(data) {
		link, err := url.Parse(value)
		if err != nil || !download(link) || link.RawQuery != "" {
			continue
		}
		match := adobeInstaller.FindStringSubmatch(link.Path)
		if match == nil {
			continue
		}
		version := match[1] + "." + match[2]
		if match[3] != "ACCCx"+strings.ReplaceAll(version, ".", "_")+".dmg" {
			continue
		}
		var numbers []int
		for part := range strings.SplitSeq(version, ".") {
			// Nine digits always fit an int.
			number, _ := strconv.Atoi(part)
			numbers = append(numbers, number)
		}
		if slices.Compare(numbers, newest) > 0 {
			newest = numbers
			release = Release{URL: link.String(), Filename: match[3], Version: version}
		}
	}
	if newest == nil {
		return Release{}, errors.New("adobe: page links no Apple silicon Creative Cloud installer")
	}
	return release, nil
}

// attributeValues reports the decoded attribute values of an HTML page's
// elements. Comments, text and script content hold URLs no reader follows.
func attributeValues(page []byte) []string {
	var values []string
	tokens := html.NewTokenizer(bytes.NewReader(page))
	for {
		token := tokens.Next()
		if token == html.ErrorToken {
			return values
		}
		if token != html.StartTagToken && token != html.SelfClosingTagToken {
			continue
		}
		_, more := tokens.TagName()
		for more {
			var value []byte
			_, value, more = tokens.TagAttr()
			values = append(values, string(value))
		}
	}
}
