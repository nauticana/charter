package binding

import (
	"context"
	"errors"
	"fmt"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

var (
	ErrNoBinding        = errors.New("no binding realizes the capability")
	ErrAmbiguousBinding = errors.New("several bindings realize the capability; name the system profile")
)

// Lookup finds the bindings that realize a capability for a system profile.
type Lookup struct {
	Provider Provider
}

// CapabilityBindingFor returns the one capability binding for the capability and profile; without a profile the capability must have exactly one binding.
func (l Lookup) CapabilityBindingFor(ctx context.Context, ownerNamespace string, capability model.Ref, profile *model.Ref) (model.CapabilityBinding, error) {
	all, err := l.Provider.CapabilityBindings(ctx)
	if err != nil {
		return model.CapabilityBinding{}, err
	}
	var matches []model.CapabilityBinding
	for _, b := range all {
		if l.realizes(b.Namespace, b.CapabilityID, b.SystemProfileID, ownerNamespace, capability, profile) {
			matches = append(matches, b)
		}
	}
	return one(matches, capability)
}

// AuthorityBindingFor returns the one authority binding for the capability and profile, when declared.
func (l Lookup) AuthorityBindingFor(ctx context.Context, ownerNamespace string, capability model.Ref, profile *model.Ref) (model.AuthorityBinding, error) {
	all, err := l.Provider.AuthorityBindings(ctx)
	if err != nil {
		return model.AuthorityBinding{}, err
	}
	var matches []model.AuthorityBinding
	for _, b := range all {
		if l.realizes(b.Namespace, b.CapabilityID, b.SystemProfileID, ownerNamespace, capability, profile) {
			matches = append(matches, b)
		}
	}
	return one(matches, capability)
}

func (Lookup) realizes(bindingNamespace string, bound, boundProfile model.Ref, owner string, capability model.Ref, profile *model.Ref) bool {
	if corpus.KeyOf(bindingNamespace, bound) != corpus.KeyOf(owner, capability) {
		return false
	}
	return profile == nil || corpus.KeyOf(bindingNamespace, boundProfile) == corpus.KeyOf(owner, *profile)
}

func one[T any](matches []T, capability model.Ref) (T, error) {
	var zero T
	switch len(matches) {
	case 0:
		return zero, fmt.Errorf("%w: %s", ErrNoBinding, capability.ID)
	case 1:
		return matches[0], nil
	}
	return zero, fmt.Errorf("%w: %s", ErrAmbiguousBinding, capability.ID)
}
