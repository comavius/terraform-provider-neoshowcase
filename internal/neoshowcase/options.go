package neoshowcase

import "net/http"

type Options struct {
	Endpoint           string
	SessionCookie      string
	InsecureSkipVerify bool
	HTTPClient         *http.Client
}
