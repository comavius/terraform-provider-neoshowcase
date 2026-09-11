package systeminfo

import (
	"testing"

	"github.com/traP-jp/terraform-provider-neoshowcase/internal/neoshowcase/gen"
)

func TestFlattenSystemInfo(t *testing.T) {
	t.Parallel()

	state, diagnostics := flattenSystemInfo(&gen.SystemInfo{
		PublicKey: "ssh-ed25519 example",
		Ssh:       &gen.SSHInfo{Host: "ssh.example.com", Port: 2222},
		Domains: []*gen.AvailableDomain{{
			Domain:         "*.example.com",
			ExcludeDomains: []string{"reserved.example.com"},
			AuthAvailable:  true,
		}},
		Ports: []*gen.AvailablePort{{
			StartPort: 30000,
			EndPort:   30100,
			Protocol:  gen.PortPublicationProtocol_TCP,
		}},
		AdditionalLinks: []*gen.AdditionalLink{{Name: "Dashboard", Url: "https://example.com"}},
		Version:         "v1.2.3",
		Revision:        "abcdef",
	})
	if diagnostics.HasError() {
		t.Fatalf("flattenSystemInfo() diagnostics = %v", diagnostics)
	}

	if got, want := state.PublicKey.ValueString(), "ssh-ed25519 example"; got != want {
		t.Errorf("public key = %q, want %q", got, want)
	}
	if state.SSH.IsNull() || state.SSH.IsUnknown() {
		t.Error("SSH object should be known and non-null")
	}
	if got, want := len(state.Domains.Elements()), 1; got != want {
		t.Errorf("domain count = %d, want %d", got, want)
	}
	if got, want := len(state.Ports.Elements()), 1; got != want {
		t.Errorf("port count = %d, want %d", got, want)
	}
	if got, want := len(state.AdditionalLinks.Elements()), 1; got != want {
		t.Errorf("link count = %d, want %d", got, want)
	}
}

func TestFlattenSystemInfoWithoutSSH(t *testing.T) {
	t.Parallel()

	state, diagnostics := flattenSystemInfo(&gen.SystemInfo{})
	if diagnostics.HasError() {
		t.Fatalf("flattenSystemInfo() diagnostics = %v", diagnostics)
	}
	if !state.SSH.IsNull() {
		t.Error("SSH object should be null when NeoShowcase does not report it")
	}
}
