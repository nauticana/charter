package binding

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

func packageJSON() []byte {
	return []byte(`{
  "formatVersion": 1,
  "id": "PKG-RESERVATION",
  "version": "1.0.0",
  "capabilityId": {"namespace":"charter.example","id":"CAP-RESERVE"},
  "contractVersion": "1.2",
  "adapterRef": "adapter:sap-reservation-v1",
  "observerRef": "observer:sap-reservation-v1",
  "parameterSchema": {
    "type":"object",
    "properties":{"operation":{"type":"string","minLength":1}},
    "required":["operation"],
    "additionalProperties":false
  },
  "bindingTemplates": [{
    "document": {
      "kind":"CapabilityBinding",
      "id":"BIND-RESERVATION",
      "bindingVersion":"1",
      "operation":"replace-me",
      "featureSupport":[{"feature":"reservation-create","support":"supported"}],
      "mappings":{"inputs":"direct","outputs":"direct","businessErrors":"mapped","authorityChecks":"external","idempotency":"key","evidence":"response"}
    },
    "parameters":{"/operation":"operation"}
  }],
  "features":[{"feature":"reservation-create","support":"supported"}],
  "lossyMappings":[],
  "conformanceCases":["reservation-create"]
}`)
}

func TestPackageManifestMaterializesCustomerBindings(t *testing.T) {
	manifest, err := ParsePackageManifest(packageJSON())
	if err != nil {
		t.Fatal(err)
	}
	profile := model.SystemProfile{Envelope: model.Envelope{CharterSpecVersion: "1.2.0", Namespace: "customer.example", ID: "PROFILE-SAP", Kind: model.KindSystemProfile}}
	docs, err := Materialize(manifest, profile, map[string]any{"operation": "POST /reservations"})
	if err != nil || len(docs) != 1 {
		t.Fatalf("materialize = %d documents, %v", len(docs), err)
	}
	binding, err := corpus.Decode[model.CapabilityBinding](docs[0])
	if err != nil {
		t.Fatal(err)
	}
	if binding.Namespace != profile.Namespace || binding.SystemProfileID.ID != profile.ID || binding.Operation != "POST /reservations" ||
		binding.CapabilityID.Namespace != "charter.example" || binding.CapabilityID.ID != "CAP-RESERVE" {
		t.Fatalf("binding = %+v", binding)
	}
	if manifest.AdapterRef == "" || manifest.ObserverRef == "" || manifest.ContractVersion != "1.2" {
		t.Fatalf("package composition metadata was lost: %+v", manifest)
	}
}

func TestPackageManifestFailsClosed(t *testing.T) {
	manifest, err := ParsePackageManifest(packageJSON())
	if err != nil {
		t.Fatal(err)
	}
	profile := model.SystemProfile{Envelope: model.Envelope{CharterSpecVersion: "1.2.0", Namespace: "customer.example", ID: "PROFILE-SAP", Kind: model.KindSystemProfile}}
	if _, err := Materialize(manifest, profile, nil); !errors.Is(err, ErrPackageManifest) {
		t.Fatalf("missing parameter: %v", err)
	}
	if _, err := Materialize(manifest, profile, map[string]any{"operation": 4}); !errors.Is(err, ErrPackageManifest) {
		t.Fatalf("wrong parameter type: %v", err)
	}

	manifest.BindingTemplates[0].Parameters = map[string]string{"/capabilityId": "operation"}
	if _, err := Materialize(manifest, profile, map[string]any{"operation": "x"}); !errors.Is(err, ErrPackageManifest) {
		t.Fatalf("protected package field: %v", err)
	}
	manifest, err = ParsePackageManifest(packageJSON())
	if err != nil {
		t.Fatal(err)
	}
	manifest.BindingTemplates[0].Parameters["/mappings"] = "operation"
	manifest.BindingTemplates[0].Parameters["/mappings/inputs"] = "operation"
	if _, err := Materialize(manifest, profile, map[string]any{"operation": "x"}); !errors.Is(err, ErrPackageManifest) {
		t.Fatalf("overlapping parameter paths: %v", err)
	}

	var value map[string]any
	if err := json.Unmarshal(packageJSON(), &value); err != nil {
		t.Fatal(err)
	}
	delete(value, "adapterRef")
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParsePackageManifest(raw); !errors.Is(err, ErrPackageManifest) {
		t.Fatalf("invalid manifest: %v", err)
	}
	if _, err := ParsePackageManifest([]byte(strings.Repeat(" ", maxPackageInputBytes+1))); !errors.Is(err, ErrPackageManifest) {
		t.Fatalf("oversized manifest: %v", err)
	}
}
