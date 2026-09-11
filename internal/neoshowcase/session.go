package neoshowcase

import "github.com/traP-jp/terraform-provider-neoshowcase/internal/neoshowcase/gen"

type Session struct {
	Client      *Client
	CurrentUser *gen.User
}
