package discovery

import (
	"context"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
)

// PythonConfig selects a Python 3 branch from 3.10 onward.
type PythonConfig struct {
	Branch string `json:"branch" jsonschema:"pattern=^3\\.[1-9][0-9]+$" jsonschema_description:"Python 3 branch from 3.10 onward."`
}

// Python selects the latest stable macOS installer in the configured branch.
func Python(ctx context.Context, client *http.Client, config PythonConfig) (Release, error) {
	target := &url.URL{Scheme: "https", Host: "www.python.org", Path: "/downloads/macos/"}
	data, err := metadata(ctx, client, target, hostURL(target.Host))
	if err != nil {
		return Release{}, fmt.Errorf("python: %w", err)
	}
	pattern := regexp.MustCompile(`Python (` + regexp.QuoteMeta(config.Branch) + `\.(0|[1-9][0-9]*))\s+-`)
	patch := -1
	version := ""
	for _, match := range pattern.FindAllStringSubmatch(html.UnescapeString(string(data)), -1) {
		value, err := strconv.Atoi(match[2])
		if err == nil && value > patch {
			patch = value
			version = match[1]
		}
	}
	if version == "" {
		return Release{}, errors.New("python: no stable macOS release matches branch")
	}
	filename := "python-" + version + "-macos11.pkg"
	return Release{
		URL:      "https://www.python.org/ftp/python/" + version + "/" + filename,
		Filename: filename,
		Version:  version,
	}, nil
}
