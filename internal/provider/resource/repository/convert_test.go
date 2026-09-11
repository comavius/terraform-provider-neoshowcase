package repository

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/traP-jp/terraform-provider-neoshowcase/internal/neoshowcase/gen"
)

func TestCreateAuthRequest(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		auth      authModel
		wantBasic bool
		wantSSH   bool
		wantError bool
	}{
		"none": {
			auth: authModel{Method: types.StringValue(authMethodNone)},
		},
		"shared SSH key": {
			auth:    authModel{Method: types.StringValue(authMethodSSH)},
			wantSSH: true,
		},
		"basic": {
			auth: authModel{
				Method:   types.StringValue(authMethodBasic),
				Username: types.StringValue("git"),
				Password: types.StringValue("secret"),
			},
			wantBasic: true,
		},
		"basic without password": {
			auth: authModel{
				Method:   types.StringValue(authMethodBasic),
				Username: types.StringValue("git"),
				Password: types.StringNull(),
			},
			wantError: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			request, err := createAuthRequest(test.auth, true)
			if test.wantError {
				if err == nil {
					t.Fatal("createAuthRequest() expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("createAuthRequest() error = %v", err)
			}
			if got := request.GetBasic() != nil; got != test.wantBasic {
				t.Errorf("basic request = %t, want %t", got, test.wantBasic)
			}
			if got := request.GetSsh() != nil; got != test.wantSSH {
				t.Errorf("SSH request = %t, want %t", got, test.wantSSH)
			}
		})
	}
}

func TestFlattenRepository(t *testing.T) {
	t.Parallel()

	priorAuth := types.ObjectValueMust(authAttributeTypes, map[string]attr.Value{
		"method":              types.StringValue(authMethodBasic),
		"username":            types.StringValue("git"),
		"password_wo":         types.StringNull(),
		"password_wo_version": types.Int64Value(3),
	})
	state, diagnostics := flattenRepository(context.Background(), &gen.Repository{
		Id:         "repository-id",
		Name:       "repository",
		Url:        "https://example.com/repository.git",
		HtmlUrl:    "https://example.com/repository",
		AuthMethod: gen.Repository_BASIC,
		OwnerIds:   []string{"member", "provider"},
	}, priorAuth, "provider")
	if diagnostics.HasError() {
		t.Fatalf("flattenRepository() diagnostics = %v", diagnostics)
	}

	auth, diagnostics := state.auth(context.Background())
	if diagnostics.HasError() {
		t.Fatalf("state.auth() diagnostics = %v", diagnostics)
	}
	if got, want := auth.Username.ValueString(), "git"; got != want {
		t.Errorf("username = %q, want %q", got, want)
	}
	if got, want := auth.PasswordVersion.ValueInt64(), int64(3); got != want {
		t.Errorf("password version = %d, want %d", got, want)
	}
	if !auth.Password.IsNull() {
		t.Error("write-only password must be null in state")
	}

	additional, ownerDiagnostics := stringsFromSet(context.Background(), state.AdditionalOwnerIDs)
	if ownerDiagnostics.HasError() {
		t.Fatalf("additional owners diagnostics = %v", ownerDiagnostics)
	}
	if want := []string{"member"}; !reflect.DeepEqual(additional, want) {
		t.Errorf("additional owners = %v, want %v", additional, want)
	}
}
