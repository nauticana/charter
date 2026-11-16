package derivation

import "errors"

var (
	ErrInvalidRequest  = errors.New("invalid derivation request")
	ErrForeignDocument = errors.New("target document of a copied identifier is not derived from its source")
	ErrDerivationFixed = errors.New("derivation of a published document is fixed")
	ErrTailored        = errors.New("tailored copy is not replaced by its source")
	ErrNotDerivable    = errors.New("kind is not derived into another document set")
	ErrNoSourceVersion = errors.New("source declares no version and none is given")
	ErrNoEnterprise    = errors.New("enterprise-scoped copy needs a target enterprise")
)
