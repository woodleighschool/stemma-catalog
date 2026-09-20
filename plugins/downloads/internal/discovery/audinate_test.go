package discovery

import "testing"

func TestAudinateSelectsCurrentFullInstaller(t *testing.T) {
	const body = `<rss xmlns:sparkle="http://www.andymatuschak.org/xml-namespaces/sparkle"><channel>
	<item><enclosure url="https://downloads.audinate.com/update.delta" sparkle:deltaFrom="4.1" sparkle:version="4.2"/></item>
	<item><sparkle:shortVersionString>4.2.3</sparkle:shortVersionString><enclosure url="https://downloads.audinate.com/current.dmg" sparkle:version="423"/></item>
	<item><enclosure url="https://downloads.audinate.com/old.dmg" sparkle:version="4.1"/></item>
	</channel></rss>`
	for product, feed := range map[string]string{
		"dante-controller":        "/DanteController/appcast/DanteController-apple_silicon.xml",
		"dante-virtual-soundcard": "/DanteVirtualSoundcard/appcast/macOS/DanteVirtualSoundcard-macOS.xml",
	} {
		t.Run(product, func(t *testing.T) {
			client := fixtureClient(t, map[string]string{"https://software-updates.audinate.com" + feed: body})
			got, err := Audinate(t.Context(), client, AudinateConfig{Product: product})
			want := Release{URL: "https://downloads.audinate.com/current.dmg", Filename: "current.dmg", Version: "4.2.3"}
			if err != nil || got != want {
				t.Fatalf("release = %+v, want %+v, error = %v", got, want, err)
			}
		})
	}
}

func TestAudinateRejectsInvalidRelease(t *testing.T) {
	for name, body := range map[string]string{
		"malformed XML":   `<rss>`,
		"empty feed":      `<rss><channel/></rss>`,
		"missing version": `<rss><channel><item><enclosure url="https://downloads.audinate.com/current.dmg"/></item></channel></rss>`,
		"insecure URL":    `<rss xmlns:sparkle="http://www.andymatuschak.org/xml-namespaces/sparkle"><channel><item><enclosure url="http://downloads.audinate.com/current.dmg" sparkle:version="4.2"/></item></channel></rss>`,
		"delta only":      `<rss xmlns:sparkle="http://www.andymatuschak.org/xml-namespaces/sparkle"><channel><item><enclosure url="https://downloads.audinate.com/update.dmg" sparkle:version="4.2" sparkle:deltaFrom="4.1"/></item></channel></rss>`,
	} {
		t.Run(name, func(t *testing.T) {
			client := fixtureClient(t, map[string]string{"https://software-updates.audinate.com/DanteController/appcast/DanteController-apple_silicon.xml": body})
			if _, err := Audinate(t.Context(), client, AudinateConfig{Product: "dante-controller"}); err == nil {
				t.Fatal("accepted invalid release")
			}
		})
	}
}
