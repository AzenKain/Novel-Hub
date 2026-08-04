package kobo

import (
	_ "embed"
	"maps"
	"strings"

	"novelhub/pkg/jsonx"
)

// kobo_resources.json is the resource map a Kobo device fetches from /v1/initialization and
// then uses to derive every other URL it calls. It is data, not code: 147 upstream Kobo store
// endpoints, copied verbatim from calibre-web's NATIVE_KOBO_RESOURCES() fallback. Embedded
// rather than transcribed into a Go literal because retyping 147 URLs by hand is a typo
// waiting to happen, and a wrong URL here breaks a device feature silently.
//
//go:embed kobo_resources.json
var resourcesJSON []byte

// Parsed once at init, and a parse failure panics. The bytes are baked in at build time, so
// the only way this can fail is a malformed commit — which then fails on the first developer
// to build, not on a user's device. Falling back to an empty map would instead hand the device
// a resource map with no URLs in it, which looks like a working sync that quietly does nothing.
//
// Values are `any`, not string: two entries (blackstone_header, free_books_page) are nested
// objects rather than URLs. They are passed through untouched.
var baseResources = func() map[string]any {
	var parsed map[string]any
	if err := jsonx.Unmarshal(resourcesJSON, &parsed); err != nil {
		panic("kobo: kobo_resources.json is malformed: " + err.Error())
	}
	return parsed
}()

// Resources returns the initialization payload for one device, with the four self-hosted keys
// rewritten to point at this server. Those four are the only ones calibre-web redirects;
// everything else stays aimed at the real Kobo store, which is what keeps store features (shop,
// wishlist, account pages) working on a device synced against a self-hosted library.
//
// endpointURL is the full token-bearing prefix the device was configured with
// (https://host/kobo/<token>), because the device appends paths to it and has no other way to
// present the token. The placeholders — {ImageId}, {width}, {height}, {Quality} — are left
// literal on purpose: the device substitutes them itself. calibre-web has to unquote() its
// generated URLs for exactly this reason; building the strings directly avoids that step.
func Resources(endpointURL string) map[string]any {
	endpointURL = strings.TrimRight(strings.TrimSpace(endpointURL), "/")

	out := maps.Clone(baseResources)

	// image_host is the origin only; the device joins it with the templates below. Derived by
	// trimming the /kobo/<token> suffix so a reverse-proxied deployment keeps its own scheme
	// and host.
	if idx := strings.Index(endpointURL, "/kobo/"); idx > 0 {
		out["image_host"] = endpointURL[:idx]
	} else {
		out["image_host"] = endpointURL
	}
	out["image_url_template"] = endpointURL + "/{ImageId}/{width}/{height}/false/image.jpg"
	out["image_url_quality_template"] = endpointURL + "/{ImageId}/{width}/{height}/{Quality}/isGreyscale/image.jpg"
	out["library_sync"] = endpointURL + "/v1/library/sync"
	return out
}
