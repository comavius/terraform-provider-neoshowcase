package ownership

import (
	"fmt"
	"slices"
)

func Effective(authoritativeOwnerID string, additionalOwnerIDs []string) []string {
	owners := make([]string, 0, len(additionalOwnerIDs)+1)
	seen := make(map[string]struct{}, len(additionalOwnerIDs)+1)

	appendOwner := func(ownerID string) {
		if ownerID == "" {
			return
		}
		if _, exists := seen[ownerID]; exists {
			return
		}
		seen[ownerID] = struct{}{}
		owners = append(owners, ownerID)
	}

	appendOwner(authoritativeOwnerID)
	for _, ownerID := range additionalOwnerIDs {
		appendOwner(ownerID)
	}
	slices.Sort(owners)
	return owners
}

func Additional(authoritativeOwnerID string, effectiveOwnerIDs []string) []string {
	owners := make([]string, 0, len(effectiveOwnerIDs))
	seen := make(map[string]struct{}, len(effectiveOwnerIDs))
	for _, ownerID := range effectiveOwnerIDs {
		if ownerID == "" || ownerID == authoritativeOwnerID {
			continue
		}
		if _, exists := seen[ownerID]; exists {
			continue
		}
		seen[ownerID] = struct{}{}
		owners = append(owners, ownerID)
	}
	slices.Sort(owners)
	return owners
}

func ValidateAuthoritativeOwner(authoritativeOwnerID string, effectiveOwnerIDs []string) error {
	if authoritativeOwnerID == "" {
		return fmt.Errorf("authoritative owner ID must not be empty")
	}
	if slices.Contains(effectiveOwnerIDs, authoritativeOwnerID) {
		return nil
	}
	return fmt.Errorf("provider user %q is not an owner; grant ownership before managing this resource", authoritativeOwnerID)
}
