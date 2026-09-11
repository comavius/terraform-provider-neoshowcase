package application

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestEnvironmentVariableKeyValidator(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		key       string
		wantError bool
	}{
		"ordinary key":         {key: "TOKEN"},
		"NS without suffix":    {key: "NS"},
		"NS without separator": {key: "NSFOO"},
		"reserved prefix only": {key: "NS_", wantError: true},
		"reserved system key":  {key: "NS_MARIADB_PASSWORD", wantError: true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			request := validator.StringRequest{
				ConfigValue: types.StringValue(test.key),
				Path:        path.Root("environment_variables").AtMapKey(test.key),
			}
			var response validator.StringResponse

			environmentVariableKeyValidator{}.ValidateString(context.Background(), request, &response)

			if got := response.Diagnostics.HasError(); got != test.wantError {
				t.Fatalf("ValidateString() error = %t, want %t; diagnostics: %v", got, test.wantError, response.Diagnostics)
			}
		})
	}
}
