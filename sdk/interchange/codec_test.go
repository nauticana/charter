package interchange

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/nauticana/charter/schema"
	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

func canonical(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// transport serializes and reparses, as an exchanged file would be.
func transport(t *testing.T, v any) map[string]any {
	t.Helper()
	var out map[string]any
	dec := json.NewDecoder(bytes.NewReader([]byte(canonical(t, v))))
	dec.UseNumber()
	if err := dec.Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestEveryDocumentSetRoundTrips(t *testing.T) {
	codec, err := New()
	if err != nil {
		t.Fatal(err)
	}
	dirs := map[string]bool{"../../examples/harbor-manufacturing/instances": true, "../../conformance/claims": true}
	if err := filepath.WalkDir("../../conformance/fixtures", func(path string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(path, ".json") {
			dirs[filepath.Dir(path)] = true
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	for dir := range dirs {
		c, err := corpus.NewDirLoader(dir).Load()
		if err != nil {
			t.Fatalf("%s: %v", dir, err)
		}
		docs := c.Documents()
		back, loss, err := codec.Import(transport(t, codec.Export(docs)), nil)
		if err != nil || len(loss) != 0 || len(back) != len(docs) {
			t.Fatalf("%s: %d of %d documents, loss %v, %v", dir, len(back), len(docs), loss, err)
		}
		for i, d := range docs {
			if got, want := canonical(t, back[i].Value), canonical(t, d.Value); got != want {
				t.Errorf("%s %s:\n got %s\nwant %s", dir, d.ID, got, want)
			}
		}
	}
}

func TestExportIdentityAndReferences(t *testing.T) {
	codec, err := New()
	if err != nil {
		t.Fatal(err)
	}
	c, err := corpus.NewDirLoader("../../examples/harbor-manufacturing/instances").Load()
	if err != nil {
		t.Fatal(err)
	}
	graph := codec.Export(c.Documents())["@graph"].([]any)
	nodes := map[string]map[string]any{}
	for _, n := range graph {
		node := n.(map[string]any)
		nodes[node["@id"].(string)] = node
	}
	act := nodes[DocumentIRI("harbor.example", "ACT-0042-EXECUTE-1")]
	if act["@type"] != "ActionRecord" || act["namespace"] != nil || act["kind"] != nil {
		t.Errorf("document node: %v", act)
	}
	if ref := act["capabilityId"].(map[string]any)["@id"]; ref != "urn:charter:harbor.example:CAP-RESERVE-ORDER-STOCK" {
		t.Errorf("idRef: %v", ref)
	}
	if actor := act["actor"].(map[string]any); actor["@id"] != "urn:charter:harbor.example:AGENT-ORDER-EXCEPTION-COORDINATOR" || actor["@type"] != "AgentIdentity" {
		t.Errorf("objectRef: %v", actor)
	}
	fact := nodes[DocumentIRI("harbor.example", "EVR-0042-STOCK-OBSERVED")]["fact"].(map[string]any)
	if subject := fact["subject"].(map[string]any); subject["@id"] != "urn:charter:external:SalesOrderItem::ORD-2026-0173-10" {
		t.Errorf("external objectRef: %v", subject)
	}
	if prop := nodes[DocumentIRI("harbor.example", "INFO-ORDER-EXCEPTION")+"/creditStatus"]; prop == nil ||
		prop["rdfs:range"].(map[string]any)["@id"] != DocumentIRI("harbor.example", "INFO-CREDIT-STATUS-SUMMARY") {
		t.Errorf("shape relationship: %v", prop)
	}
	ns, id, ok := ParseDocumentIRI(DocumentIRI("https://harbor.example/charter", "POS:1"))
	if !ok || ns != "https://harbor.example/charter" || id != "POS:1" {
		t.Errorf("identity with a URI namespace: %q %q %t", ns, id, ok)
	}
	if _, _, ok := ParseDocumentIRI("urn:charter:external:Order::O-1"); ok {
		t.Error("external IRI parsed as a document")
	}
}

func TestPublishedContextIsCurrent(t *testing.T) {
	codec, err := New()
	if err != nil {
		t.Fatal(err)
	}
	b, err := fs.ReadFile(schema.FS, "context.jsonld")
	if err != nil {
		t.Fatal(err)
	}
	var published any
	if err := json.Unmarshal(b, &published); err != nil {
		t.Fatal(err)
	}
	var generated any
	if err := json.Unmarshal([]byte(canonical(t, codec.Context())), &generated); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(published, generated) {
		t.Error("schema/context.jsonld differs from the generated context; run charter context > schema/context.jsonld")
	}
	terms := generated.(map[string]any)["@context"].(map[string]any)
	if terms["@vocab"] != Vocab || terms["capabilityId"].(map[string]any)["@type"] != "@id" || terms["recordedAt"].(map[string]any)["@type"] != "xsd:dateTime" ||
		terms["from"].(map[string]any)["@type"] != "xsd:date" || terms["locator"].(map[string]any)["@type"] != "@id" {
		t.Errorf("context terms: %v", terms)
	}
}

func TestLossIsRefusedUnlessAccepted(t *testing.T) {
	codec, err := New()
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("testdata/lossy.jsonld")
	if err != nil {
		t.Fatal(err)
	}
	graph := transport(t, json.RawMessage(b))
	_, loss, err := codec.Import(graph, nil)
	want := []string{"@language of description", "node _:b0", "reference https://other.example/positions/7", "term headcount", "term note"}
	if !errors.Is(err, ErrLossy) || !reflect.DeepEqual(loss, want) {
		t.Fatalf("loss %v, %v", loss, err)
	}
	docs, loss, err := codec.Import(graph, want)
	if err != nil || len(docs) != 1 || !reflect.DeepEqual(loss, want) {
		t.Fatalf("accepted loss: %d documents, %v, %v", len(docs), loss, err)
	}
	ou, err := corpus.Decode[model.OrganizationUnit](docs[0])
	if err != nil || ou.ID != "OU-1" || ou.Kind != model.KindOrganizationUnit || ou.Description != "Vertrieb" || ou.ParentUnitID != nil {
		t.Errorf("imported %+v %v", ou, err)
	}
	locator := transport(t, map[string]any{"@graph": []any{map[string]any{"@id": "urn:charter:t:EVR-1", "@type": "EvidenceRecord", "charterSpecVersion": "1.8.0",
		"category": "observed-fact", "content": "c", "recordedAt": "2026-01-01T00:00:00Z", "sources": []any{map[string]any{"locator": map[string]any{"@id": "https://docs.example/a.md#s1"}}}}}})
	if evr, loss, err := codec.Import(locator, nil); err != nil || len(loss) != 0 || evr[0].Value.(map[string]any)["sources"].([]any)[0].(map[string]any)["locator"] != "https://docs.example/a.md#s1" {
		t.Errorf("locator as a node reference: %v %v", loss, err)
	}
	if value := docs[0].Value.(map[string]any); value["headcount"] != nil || value["validity"].(map[string]any)["note"] != nil || len(value["validity"].(map[string]any)) != 1 {
		t.Errorf("lost terms were written: %v", value)
	}
	if string(ou.Extensions["https://other.example/vocab#costCenter"]) != `"CC-1200"` {
		t.Errorf("foreign property not kept as an extension: %s", ou.Extensions)
	}
}
