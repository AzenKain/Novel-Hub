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

var baseResources = func() map[string]any {
	var parsed map[string]any
	if err := jsonx.Unmarshal(resourcesJSON, &parsed); err != nil {
		panic("kobo: kobo_resources.json is malformed: " + err.Error())
	}
	return parsed
}()

// Resources returns the initialization payload for one device, with the four self-hosted keys rewritten to point at this server.
func Resources(endpointURL string) map[string]any {
	endpointURL = strings.TrimRight(strings.TrimSpace(endpointURL), "/")

	out := maps.Clone(baseResources)

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
