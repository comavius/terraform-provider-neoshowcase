package application

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

const reservedEnvironmentVariablePrefix = "NS_"

var _ validator.String = environmentVariableKeyValidator{}

type environmentVariableKeyValidator struct{}

func (environmentVariableKeyValidator) Description(context.Context) string {
	return "environment variable keys must not begin with the NeoShowcase-reserved NS_ prefix"
}

func (v environmentVariableKeyValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v environmentVariableKeyValidator) ValidateString(ctx context.Context, request validator.StringRequest, response *validator.StringResponse) {
	if request.ConfigValue.IsNull() || request.ConfigValue.IsUnknown() {
		return
	}

	if strings.HasPrefix(request.ConfigValue.ValueString(), reservedEnvironmentVariablePrefix) {
		response.Diagnostics.Append(diag.NewAttributeErrorDiagnostic(
			request.Path,
			"Reserved environment variable key",
			v.Description(ctx),
		))
	}
}
