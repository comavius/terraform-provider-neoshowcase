package ownership

import (
	"reflect"
	"testing"
)

func TestEffective(t *testing.T) {
	t.Parallel()

	got := Effective("provider", []string{"member-b", "provider", "member-a", "member-b", ""})
	want := []string{"member-a", "member-b", "provider"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Effective() = %v, want %v", got, want)
	}
}

func TestAdditional(t *testing.T) {
	t.Parallel()

	got := Additional("provider", []string{"member-b", "provider", "member-a", "member-b"})
	want := []string{"member-a", "member-b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Additional() = %v, want %v", got, want)
	}
}

func TestValidateAuthoritativeOwner(t *testing.T) {
	t.Parallel()

	if err := ValidateAuthoritativeOwner("provider", []string{"provider"}); err != nil {
		t.Fatalf("owner should be accepted: %v", err)
	}
	if err := ValidateAuthoritativeOwner("provider", []string{"member"}); err == nil {
		t.Fatal("non-owner should be rejected")
	}
}
