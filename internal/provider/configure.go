package provider

import (
	"context"
	"os"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/provider"

	"github.com/traP-jp/terraform-provider-neoshowcase/internal/neoshowcase"
)

const defaultEndpoint = "https://ns.trap.jp"

func (p *neoShowcaseProvider) Configure(ctx context.Context, request provider.ConfigureRequest, response *provider.ConfigureResponse) {
	var config providerModel
	response.Diagnostics.Append(request.Config.Get(ctx, &config)...)
	if response.Diagnostics.HasError() {
		return
	}

	endpoint, _ := configuredString(config.Endpoint.ValueString(), os.Getenv("NEOSHOWCASE_ENDPOINT"), defaultEndpoint)
	if config.Endpoint.IsUnknown() {
		response.Diagnostics.AddError("Unknown NeoShowcase endpoint", "The endpoint provider attribute must be known during provider configuration.")
	}
	sessionCookie := strings.TrimSpace(os.Getenv("NEOSHOWCASE_SESSION_COOKIE"))
	if sessionCookie == "" {
		response.Diagnostics.AddError("Missing NeoShowcase session cookie", "Set the NEOSHOWCASE_SESSION_COOKIE environment variable.")
	}
	if response.Diagnostics.HasError() {
		return
	}

	client, err := neoshowcase.NewClient(neoshowcase.Options{
		Endpoint:           endpoint,
		SessionCookie:      sessionCookie,
		InsecureSkipVerify: config.InsecureSkipVerify.ValueBool(),
	})
	if err != nil {
		response.Diagnostics.AddError("Invalid NeoShowcase client configuration", err.Error())
		return
	}

	currentUser, err := client.GetMe(ctx)
	if err != nil {
		response.Diagnostics.AddError("Unable to configure NeoShowcase client", err.Error())
		return
	}
	if currentUser.GetId() == "" {
		response.Diagnostics.AddError("Invalid NeoShowcase user", "GetMe returned an empty user ID.")
		return
	}

	session := &neoshowcase.Session{Client: client, CurrentUser: currentUser}
	response.DataSourceData = session
	response.ResourceData = session
}

func configuredString(configured, environment, fallback string) (string, bool) {
	for _, value := range []string{configured, environment, fallback} {
		if value = strings.TrimSpace(value); value != "" {
			return value, true
		}
	}
	return "", false
}
