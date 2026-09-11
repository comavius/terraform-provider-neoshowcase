package neoshowcase

import "net/http"

const DefaultAuthHeader = "X-Showcase-User"

type Options struct {
	Endpoint           string
	User               string
	AuthHeader         string
	AdditionalHeaders  map[string]string
	InsecureSkipVerify bool
	HTTPClient         *http.Client
}
