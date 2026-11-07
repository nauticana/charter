// Package interchange exchanges Charter document sets as JSON-LD under the published Charter context (CHR-CONF-016).
package interchange

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"sort"
	"strings"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/validate"
)

const (
	Vocab      = "https://nauticana.github.io/charter/vocab#"
	ContextIRI = "https://nauticana.github.io/charter/schema/context.jsonld"
	urnPrefix  = "urn:charter:"
	externalNS = "external"
	xsd        = "http://www.w3.org/2001/XMLSchema#"
)

var ErrLossy = errors.New("import would lose content the caller has not accepted")

// Codec maps Charter documents to compact JSON-LD and back, driven by the embedded schemas.
type Codec struct{ meta *validate.SchemaMeta }

func New() (*Codec, error) {
	meta, err := validate.NewSchemaMeta()
	if err != nil {
		return nil, err
	}
	return &Codec{meta: meta}, nil
}

// DocumentIRI is the stable identity of a document: the namespace percent-encoded, so the identifier follows the last colon.
func DocumentIRI(namespace, id string) string {
	return urnPrefix + url.QueryEscape(namespace) + ":" + id
}

// ParseDocumentIRI inverts DocumentIRI; external object IRIs and other IRIs are refused.
func ParseDocumentIRI(iri string) (namespace, id string, ok bool) {
	ns, rest, found := strings.Cut(strings.TrimPrefix(iri, urnPrefix), ":")
	if !strings.HasPrefix(iri, urnPrefix) || !found || ns == externalNS || rest == "" {
		return "", "", false
	}
	namespace, err := url.QueryUnescape(ns)
	return namespace, rest, err == nil
}

// Context is the JSON-LD context document: every Charter term under the vocabulary, references typed as IRIs, and
// dates typed as xsd values.
func (c *Codec) Context() map[string]any {
	terms := map[string]any{"@version": 1.1, "@vocab": Vocab, "chr": Vocab, "xsd": xsd,
		"rdf": "http://www.w3.org/1999/02/22-rdf-syntax-ns#", "rdfs": "http://www.w3.org/2000/01/rdf-schema#"}
	for k := range c.meta.IDRefKeys {
		terms[k] = map[string]any{"@id": "chr:" + k, "@type": "@id"}
	}
	for k, format := range c.meta.Formats {
		if t, ok := map[string]string{"date": "xsd:date", "date-time": "xsd:dateTime"}[format]; ok && !c.meta.IDRefKeys[k] {
			terms[k] = map[string]any{"@id": "chr:" + k, "@type": t}
		}
	}
	return map[string]any{"@context": terms}
}

// Export renders documents as one JSON-LD graph under the published context, followed by the RDF properties the
// information shapes declare.
func (c *Codec) Export(docs []*model.Document) map[string]any {
	graph := make([]any, 0, len(docs))
	for _, d := range docs {
		value, _ := d.Value.(map[string]any)
		node := map[string]any{"@id": DocumentIRI(d.Namespace, d.ID), "@type": string(d.Kind)}
		for k, v := range value {
			switch k {
			case "namespace", "id", "kind":
			case "extensions":
				ext, _ := v.(map[string]any)
				for ek, ev := range ext {
					node[ek] = ev
				}
			default:
				node[k] = c.export(d.Namespace, k, v)
			}
		}
		graph = append(graph, node)
		if d.Kind == model.KindInformationDefinition {
			graph = append(graph, shapeProperties(d)...)
		}
	}
	return map[string]any{"@context": ContextIRI, "@graph": graph}
}

func (c *Codec) export(ns, key string, v any) any {
	switch n := v.(type) {
	case []any:
		out := make([]any, len(n))
		for i, x := range n {
			out[i] = c.export(ns, key, x)
		}
		return out
	case map[string]any:
		if kind, isObject := n["kind"].(string); c.meta.ObjectRefKeys[key] && isObject {
			id, _ := n["id"].(string)
			target, _ := n["namespace"].(string)
			if external, _ := n["external"].(bool); external {
				return map[string]any{"@id": urnPrefix + externalNS + ":" + url.QueryEscape(kind) + ":" + url.QueryEscape(target) + ":" + id, "@type": kind}
			}
			return map[string]any{"@id": DocumentIRI(or(target, ns), id), "@type": kind}
		}
		if id, isRef := n["id"].(string); c.meta.IDRefKeys[key] && isRef && len(n) == 2 {
			target, _ := n["namespace"].(string)
			return map[string]any{"@id": DocumentIRI(target, id)}
		}
		out := make(map[string]any, len(n))
		for k, x := range n {
			out[k] = c.export(ns, k, x)
		}
		return out
	case string:
		if c.meta.IDRefKeys[key] {
			return map[string]any{"@id": DocumentIRI(ns, n)}
		}
	}
	return v
}

// shapeProperties states each shape property and relationship as an RDF property of the definition's class.
func shapeProperties(d *model.Document) []any {
	def, err := corpus.Decode[model.InformationDefinition](d)
	if err != nil || def.Shape == nil {
		return nil
	}
	class := DocumentIRI(d.Namespace, d.ID)
	var out []any
	property := func(name string, rng string) {
		out = append(out, map[string]any{"@id": class + "/" + name, "@type": "rdf:Property", "rdfs:label": name,
			"rdfs:domain": map[string]any{"@id": class}, "rdfs:range": map[string]any{"@id": rng}})
	}
	for _, p := range def.Shape.Properties {
		property(p.Name, "xsd:"+or(map[string]string{"integer": "integer", "number": "decimal", "boolean": "boolean"}[p.Type], "string"))
	}
	for _, r := range def.Shape.Relationships {
		property(r.Name, DocumentIRI(or(r.TargetInformationID.Namespace, d.Namespace), r.TargetInformationID.ID))
	}
	return out
}

// Import reads a graph in the compact form Export writes. Content Charter cannot carry is loss: a term the vocabulary
// lacks, a node of no Charter kind, a blank node, a reference outside the Charter identity scheme, or a JSON-LD keyword
// beyond the identity, type, value, and declared datatype of a value. Every loss is returned and omitted; one the
// caller did not list in accepted refuses the import (CHR-CONF-016).
func (c *Codec) Import(graph map[string]any, accepted []string) ([]*model.Document, []string, error) {
	nodes, _ := graph["@graph"].([]any)
	if nodes == nil {
		nodes = []any{graph}
	}
	in := &importer{Codec: c}
	var docs []*model.Document
	for _, n := range nodes {
		node, _ := n.(map[string]any)
		iri, _ := node["@id"].(string)
		kind := c.kindOf(node["@type"])
		ns, id, ok := ParseDocumentIRI(iri)
		switch {
		case kind == "rdf:Property":
			continue
		case !ok || !c.meta.Kinds[kind]:
			in.lose("node " + or(iri, "without @id"))
			continue
		}
		value := map[string]any{"namespace": ns, "id": id, "kind": kind}
		in.ns = ns
		for k, v := range node {
			switch term := c.term(k); {
			case strings.HasPrefix(k, "@"):
			case strings.HasPrefix(term, "?"):
				in.lose("term " + term[1:])
			case term == "":
				ext, _ := value["extensions"].(map[string]any)
				if ext == nil {
					ext = map[string]any{}
				}
				ext[k] = v
				value["extensions"] = ext
			default:
				value[term] = in.value(term, v)
			}
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, nil, err
		}
		d, err := (corpus.Parser{}).Parse(raw)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %w", iri, err)
		}
		docs = append(docs, d)
	}
	sort.Strings(in.loss)
	in.loss = slices.Compact(in.loss)
	for _, l := range in.loss {
		if !slices.Contains(accepted, l) {
			return nil, in.loss, fmt.Errorf("%w: %s", ErrLossy, strings.Join(in.loss, "; "))
		}
	}
	return docs, in.loss, nil
}

type importer struct {
	*Codec
	ns   string
	loss []string
}

func (in *importer) lose(what string) { in.loss = append(in.loss, what) }

// term maps a key to the Charter property it names: a plain vocabulary term, or one spelled as a CURIE or full IRI.
// Empty means the key belongs to another vocabulary and is kept as an extension; an unknown plain term is loss.
func (c *Codec) term(key string) string {
	name := strings.TrimPrefix(strings.TrimPrefix(key, Vocab), "chr:")
	if c.meta.Properties[name] {
		return name
	}
	if strings.ContainsAny(key, ":/") {
		return ""
	}
	return "?" + key
}

func (c *Codec) kindOf(t any) string {
	if list, ok := t.([]any); ok && len(list) == 1 {
		t = list[0]
	}
	s, _ := t.(string)
	return strings.TrimPrefix(strings.TrimPrefix(s, Vocab), "chr:")
}

func (in *importer) value(term string, v any) any {
	switch n := v.(type) {
	case []any:
		out := make([]any, len(n))
		for i, x := range n {
			out[i] = in.value(term, x)
		}
		return out
	case map[string]any:
		if literal, ok := n["@value"]; ok {
			for key, value := range n {
				if key == "@value" {
					continue
				}
				if key == "@type" && in.datatype(term, value) {
					continue
				}
				in.lose(key + " of " + term)
			}
			return literal
		}
		if iri, ok := n["@id"].(string); ok {
			for key := range n {
				if key != "@id" && key != "@type" {
					in.lose(key + " of reference " + iri)
				}
			}
			return in.reference(iri, in.kindOf(n["@type"]))
		}
		out := make(map[string]any, len(n))
		for k, x := range n {
			switch t := in.term(k); {
			case strings.HasPrefix(k, "@"):
				in.lose(k + " of " + term)
			case strings.HasPrefix(t, "?"):
				in.lose("term " + t[1:])
			case t == "":
				in.lose(k + " of " + term)
			default:
				out[t] = in.value(t, x)
			}
		}
		return out
	case string:
		if in.meta.IDRefKeys[term] {
			return in.reference(n, "")
		}
	}
	return v
}

func (in *importer) datatype(term string, value any) bool {
	typeName, _ := value.(string)
	format := in.meta.Formats[term]
	return format == "date" && (typeName == "xsd:date" || typeName == xsd+"date") ||
		format == "date-time" && (typeName == "xsd:dateTime" || typeName == xsd+"dateTime")
}

// reference turns a node reference back into an idRef or objectRef; the same namespace compacts to a bare identifier.
func (in *importer) reference(iri, kind string) any {
	if rest, ok := strings.CutPrefix(iri, urnPrefix+externalNS+":"); ok {
		parts := strings.SplitN(rest, ":", 3)
		if len(parts) != 3 {
			in.lose("reference " + iri)
			return nil
		}
		k, _ := url.QueryUnescape(parts[0])
		ns, _ := url.QueryUnescape(parts[1])
		out := map[string]any{"kind": k, "id": parts[2], "external": true}
		if ns != "" {
			out["namespace"] = ns
		}
		return out
	}
	ns, id, ok := ParseDocumentIRI(iri)
	if !ok {
		in.lose("reference " + iri)
		return nil
	}
	switch {
	case kind != "":
		out := map[string]any{"kind": kind, "id": id}
		if ns != in.ns {
			out["namespace"] = ns
		}
		return out
	case ns != in.ns:
		return map[string]any{"namespace": ns, "id": id}
	}
	return id
}

func or(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
