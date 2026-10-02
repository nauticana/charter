package evidence_test

import (
	"testing"

	"github.com/nauticana/charter/sdk/evidence"
	"github.com/nauticana/charter/sdk/evidence/evidencetest"
)

func TestBaseMemoryStoreContract(t *testing.T) {
	evidencetest.Run(t, func(*testing.T) evidence.Store { return evidence.NewBaseMemoryStore() },
		func(s evidence.Store) evidence.Provider { return evidence.NewBaseProvider(s) })
}
