package discovery

import (
	"net/http"
	"testing"
)

const (
	adobePage = "https://helpx.adobe.com/download-install/apps/download-install-apps/creative-cloud-apps/download-creative-cloud-desktop-app-using-direct-links.html"
	adobeESD  = "https://ccmdls.adobe.com/AdobeProducts/StandaloneBuilds/ACCC/ESD/"
)

func adobeClient(t *testing.T, page string) *http.Client {
	t.Helper()
	client := fixtureClient(t, map[string]string{adobePage: page})
	fixture := client.Transport
	client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		// Adobe's edge answers 403 to the plugin's own User-Agent.
		if agent := req.Header.Get("User-Agent"); agent != "" {
			t.Errorf("User-Agent = %q, want Go's default", agent)
		}
		return fixture.RoundTrip(req)
	})
	return client
}

func TestAdobeSelectsNewestLinkedAppleSiliconInstaller(t *testing.T) {
	client := adobeClient(t, `<html><head>
	<script>document.write('<a href="`+adobeESD+`9.0.0/1/macarm64/ACCCx9_0_0_1.dmg">')</script>
	</head><body><table>
	<!-- <a href="`+adobeESD+`8.0.0/1/macarm64/ACCCx8_0_0_1.dmg">Download</a> -->
	<tr><td><a href="`+adobeESD+`6.9.0/900/macarm64/ACCCx6_9_0_900.dmg">Download</a></td>
	<td><a disablelinktracking="false" href="`+adobeESD+`6.10.0/252.41/macarm64/ACCCx6_10_0_252_41.dmg" target="_blank">Download</a></td>
	<td><a href="`+adobeESD+`7.0.0/1/osx10/ACCCx7_0_0_1.dmg">Intel</a></td>
	<td><a href="`+adobeESD+`7.0.0/2/macarm64/ACCCx6_10_0_252_41.dmg">Renamed</a></td>
	<td><a href="https://ccmdls.adobe.com.example/AdobeProducts/StandaloneBuilds/ACCC/ESD/7.0.0/3/macarm64/ACCCx7_0_0_3.dmg">Lookalike</a></td>
	<td><a href="`+adobeESD+`7.0.0/4/macarm64/ACCCx7_0_0_4.dmg?token=1">Expiring</a></td>
	<td><a href="`+adobeESD+`6.10.0/252.41/macarm64/ACCCx6_10_0_252_41.dmg">Repeated</a></td></tr>
	</table></body></html>`)
	got, err := Adobe(t.Context(), client, AdobeConfig{})
	if err != nil {
		t.Fatal(err)
	}
	want := Release{URL: adobeESD + "6.10.0/252.41/macarm64/ACCCx6_10_0_252_41.dmg", Filename: "ACCCx6_10_0_252_41.dmg", Version: "6.10.0.252.41"}
	if got != want {
		t.Fatalf("release = %+v, want %+v", got, want)
	}
}

func TestAdobeRequiresAnAppleSiliconInstaller(t *testing.T) {
	client := adobeClient(t, `<a href="`+adobeESD+`6.10.0/252.41/osx10/ACCCx6_10_0_252_41.dmg">Intel</a>
	<p>`+adobeESD+`6.10.0/252.41/macarm64/ACCCx6_10_0_252_41.dmg</p>`)
	if release, err := Adobe(t.Context(), client, AdobeConfig{}); err == nil {
		t.Fatalf("accepted %+v", release)
	}
}
