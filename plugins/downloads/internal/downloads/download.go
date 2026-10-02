package downloads

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/woodleighschool/stemma-catalog/plugins/downloads/internal/discovery"
)

// Client bounds metadata requests and refuses insecure redirects. Resolvers use public
// vendor endpoints; credentials and arbitrary request headers are not accepted.
func Client() *http.Client {
	return &http.Client{
		Timeout: 15 * time.Minute,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("too many redirects")
			}
			return validURL(req.URL.String())
		},
	}
}

func validURL(address string) error {
	u, err := url.Parse(address)
	if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil || u.Fragment != "" || u.Port() != "" && u.Port() != "443" {
		return errors.New("download requires an HTTPS URL without credentials, fragments or custom ports")
	}
	return nil
}

func validateRelease(release discovery.Release) error {
	if err := validURL(release.URL); err != nil {
		return err
	}
	name := release.Filename
	if name == "" || name == "." || name == ".." || strings.ContainsAny(name, "/\\:\x00\r\n\t") || strings.TrimSpace(name) != name {
		return errors.New("download requires a safe filename")
	}
	if strings.ContainsAny(release.Version, "\x00\r\n\t") {
		return errors.New("invalid release version")
	}
	return nil
}
