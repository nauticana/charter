package derivation

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
	"github.com/nauticana/charter/sdk/validate"
)

const (
	harborDir = "../../examples/harbor-manufacturing/instances"
	src       = "harbor.example"
	dst       = "tenant.example"
	release   = "2026.1"
)

func options() Options {
	return Options{SourceNamespace: src, TargetNamespace: dst, Enterprise: model.Ref{ID: "ENT-TENANT"}, SourceVersion: release}
}

func harbor(t *testing.T) *corpus.Corpus {
	t.Helper()
	c, err := corpus.NewDirLoader(harborDir).Load()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func deriver(t *testing.T) *Deriver {
	t.Helper()
	dv, err := NewDeriver()
	if err != nil {
		t.Fatal(err)
	}
	return dv
}

// edited returns a document changed by edit, reparsed from its JSON.
func edited(t *testing.T, d *model.Document, edit func(map[string]any)) *model.Document {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal(d.Raw, &v); err != nil {
		t.Fatal(err)
	}
	edit(v)
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	out, err := (corpus.Parser{}).Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// with returns the documents of base and docs, the last document of each key winning.
func with(t *testing.T, base *corpus.Corpus, docs ...*model.Document) *corpus.Corpus {
	t.Helper()
	replaced := map[corpus.DocumentKey]*model.Document{}
	for _, d := range docs {
		replaced[corpus.DocumentKey{Namespace: d.Namespace, ID: d.ID}] = d
	}
	out := corpus.New()
	for _, d := range base.Documents() {
		key := corpus.DocumentKey{Namespace: d.Namespace, ID: d.ID}
		if r, ok := replaced[key]; ok {
			d = r
			delete(replaced, key)
		}
		if err := out.Add(d); err != nil {
			t.Fatal(err)
		}
	}
	for _, d := range docs {
		key := corpus.DocumentKey{Namespace: d.Namespace, ID: d.ID}
		if last, left := replaced[key]; left {
			delete(replaced, key)
			if err := out.Add(last); err != nil {
				t.Fatal(err)
			}
		}
	}
	return out
}

// tenant is a target set holding only its own enterprise.
func tenant(t *testing.T, source *corpus.Corpus) *corpus.Corpus {
	t.Helper()
	ent, _ := source.Get(src, "ENT-HARBOR")
	return with(t, corpus.New(), edited(t, ent, func(v map[string]any) { v["namespace"], v["id"] = dst, "ENT-TENANT" }))
}

func derive(t *testing.T, source, target corpus.Source, opts Options, roots ...Root) []*model.Document {
	t.Helper()
	out, err := deriver(t).Derive(context.Background(), source, target, roots, opts)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func ids(docs []*model.Document) map[string]*model.Document {
	out := map[string]*model.Document{}
	for _, d := range docs {
		out[d.ID] = d
	}
	return out
}

func TestDeriveAdoptsTheReferencedSourceIntoTheTarget(t *testing.T) {
	source := harbor(t)
	target := tenant(t, source)
	out := derive(t, source, target, options(), Root{ID: "TASK-OE-DETECT", Version: "1"})
	copies := ids(out)
	for _, id := range []string{"TASK-OE-DETECT", "PROC-RESOLVE-ORDER-EXCEPTION", "RESP-COORDINATE-ORDER-EXCEPTION"} {
		if copies[id] == nil {
			t.Fatalf("%s not derived: %v", id, copies)
		}
	}
	meta, err := validate.NewSchemaMeta()
	if err != nil {
		t.Fatal(err)
	}
	for i, c := range out {
		if i > 0 && out[i-1].ID >= c.ID {
			t.Fatalf("copies not ordered by identifier at %s", c.ID)
		}
		s, _ := source.Get(src, c.ID)
		digest, err := corpus.ContentDigest(s)
		if err != nil {
			t.Fatal(err)
		}
		version, versioned := corpus.DefinitionVersion(s)
		if !versioned {
			version = release
		}
		p := c.DerivedFrom
		if c.Namespace != dst || p == nil || p.Source != (model.ObjectRef{Kind: s.Kind, Namespace: src, ID: s.ID}) ||
			p.Version != version || p.Digest != digest || p.Adoption != model.AdoptionAdopted {
			t.Fatalf("%s: namespace %s, derivation %+v", c.ID, c.Namespace, p)
		}
		if c.EnterpriseID != nil && *c.EnterpriseID != (model.Ref{ID: "ENT-TENANT"}) {
			t.Fatalf("%s: enterprise %+v", c.ID, c.EnterpriseID)
		}
		meta.WalkRefs(c.Value, func(key string, ref model.Ref, _ any) {
			if ref.Namespace == src {
				t.Errorf("%s: %s still names the source", c.ID, key)
			}
		}, func(key string, ref model.ObjectRef, _ any) {
			if ref.Namespace == src {
				t.Errorf("%s: %s still names the source", c.ID, key)
			}
		})
	}
	a, err := validate.NewAdditions()
	if err != nil {
		t.Fatal(err)
	}
	findings, err := a.Validate(context.Background(), target, out...)
	if err != nil || len(findings) != 0 {
		t.Fatalf("copies do not validate in the target: %+v %v", findings, err)
	}
}

func TestDeriveNamesExplicitSourceReferencesInTheTarget(t *testing.T) {
	base := harbor(t)
	task, _ := base.Get(src, "TASK-OE-DETECT")
	source := with(t, base, edited(t, task, func(v map[string]any) {
		v["accountableResponsibilityId"] = map[string]any{"namespace": src, "id": "RESP-COORDINATE-ORDER-EXCEPTION"}
		v["processId"] = map[string]any{"namespace": "other.example", "id": "PROC-ELSEWHERE"}
	}))
	copies := ids(derive(t, source, tenant(t, source), options(), Root{ID: "TASK-OE-DETECT"}))
	c := copies["TASK-OE-DETECT"].Value.(map[string]any)
	if got := c["accountableResponsibilityId"].(map[string]any)["namespace"]; got != dst || copies["RESP-COORDINATE-ORDER-EXCEPTION"] == nil {
		t.Fatalf("source reference: %v", c["accountableResponsibilityId"])
	}
	if got := c["processId"].(map[string]any)["namespace"]; got != "other.example" || copies["PROC-RESOLVE-ORDER-EXCEPTION"] != nil {
		t.Fatalf("reference elsewhere: %v", c["processId"])
	}
}

func TestDeriveLeavesSourceEvidenceInTheSource(t *testing.T) {
	base := harbor(t)
	gap, _ := base.Get(src, "GAP-ORDER-EXCEPTION-EVIDENCE")
	elsewhere := map[string]any{"namespace": "other.example", "id": "EVR-ELSEWHERE"}
	source := with(t, base, edited(t, gap, func(v map[string]any) {
		v["sourceEvidence"] = append(v["sourceEvidence"].([]any), elsewhere)
	}))
	copies := ids(derive(t, source, tenant(t, source), options(), Root{ID: gap.ID}))
	cited := copies[gap.ID].Value.(map[string]any)["sourceEvidence"].([]any)
	if len(cited) != 1 || cited[0].(map[string]any)["id"] != "EVR-ELSEWHERE" || copies["EVR-GAP-EVIDENCE-DISCOVERY"] != nil {
		t.Fatalf("source evidence: %v", cited)
	}
	only := with(t, base, edited(t, gap, func(v map[string]any) {}))
	if _, cites := ids(derive(t, only, tenant(t, only), options(), Root{ID: gap.ID}))[gap.ID].Value.(map[string]any)["sourceEvidence"]; cites {
		t.Fatal("evidence of the source set cited by the copy")
	}
}

func TestDeriveReusesACopyAndUpgradesANewVersion(t *testing.T) {
	source := harbor(t)
	first := derive(t, source, tenant(t, source), options(), Root{ID: "TASK-OE-DETECT", Version: "1"})
	target := with(t, tenant(t, source), first...)
	if again := derive(t, source, target, options(), Root{ID: "TASK-OE-DETECT", Version: "1"}); len(again) != 0 {
		t.Fatalf("an unchanged derivation is written again: %d", len(again))
	}
	task, _ := source.Get(src, "TASK-OE-DETECT")
	next := with(t, source, edited(t, task, func(v map[string]any) {
		v["taskDefinitionVersion"], v["name"], v["accountableResponsibilityId"] = "2", "Detect faster", "RESP-INVESTIGATE-ORDER-EXCEPTION"
	}))
	if kept := derive(t, next, target, options(), Root{ID: "TASK-OE-DETECT", Version: "2"}); len(kept) != 0 {
		t.Fatalf("a kept copy is replaced or its newer references followed: %v", ids(kept))
	}
	upgrade := options()
	upgrade.Upgrade = true
	out := derive(t, next, target, upgrade, Root{ID: "TASK-OE-DETECT", Version: "2"})
	upgraded := ids(out)
	if c := upgraded["TASK-OE-DETECT"]; c == nil || c.DerivedFrom.Version != "2" || c.Name != "Detect faster" ||
		upgraded["RESP-INVESTIGATE-ORDER-EXCEPTION"] == nil || upgraded["RESP-COORDINATE-ORDER-EXCEPTION"] != nil {
		t.Fatalf("upgrade: %v", upgraded)
	}
	a, err := validate.NewAdditions()
	if err != nil {
		t.Fatal(err)
	}
	if f, err := a.Validate(context.Background(), target, out...); err != nil || len(f) != 0 {
		t.Fatalf("a new version of a published copy records its own derivation: %+v %v", f, err)
	}
}

func TestDeriveRefusesToReplaceAFixedTailoredOrForeignDocument(t *testing.T) {
	source := harbor(t)
	const id = "RESP-COORDINATE-ORDER-EXCEPTION"
	copies := derive(t, source, tenant(t, source), options(), Root{ID: id})
	copied := ids(copies)[id]
	resp, _ := source.Get(src, id)
	changed := with(t, source, edited(t, resp, func(v map[string]any) { v["name"] = "Coordinate every order exception" }))
	upgrade := options()
	upgrade.Upgrade = true
	draft := edited(t, copied, func(v map[string]any) { v["lifecycleState"] = model.LifecycleProposed })
	if out := derive(t, changed, with(t, tenant(t, source), append(copies, draft)...), upgrade, Root{ID: id}); len(out) != 1 || out[0].ID != id {
		t.Fatalf("a draft copy is not upgraded: %+v", out)
	}
	for name, tc := range map[string]struct {
		held *model.Document
		want error
	}{
		"published": {copied, ErrDerivationFixed},
		"tailored": {edited(t, copied, func(v map[string]any) {
			v["derivedFrom"].(map[string]any)["adoption"] = model.AdoptionTailored
		}), ErrTailored},
		"underived": {edited(t, copied, func(v map[string]any) { delete(v, "derivedFrom") }), ErrForeignDocument},
		"derived elsewhere": {edited(t, copied, func(v map[string]any) {
			v["derivedFrom"].(map[string]any)["source"].(map[string]any)["namespace"] = "other.example"
		}), ErrForeignDocument},
	} {
		target := with(t, tenant(t, source), append(copies, tc.held)...)
		if _, err := deriver(t).Derive(context.Background(), changed, target, []Root{{ID: id}}, upgrade); !errors.Is(err, tc.want) {
			t.Errorf("%s: %v", name, err)
		}
	}
	foreign := with(t, tenant(t, source), edited(t, copied, func(v map[string]any) { delete(v, "derivedFrom") }))
	if _, err := deriver(t).Derive(context.Background(), source, foreign, []Root{{ID: id}}, options()); !errors.Is(err, ErrForeignDocument) {
		t.Errorf("a foreign document is reused: %v", err)
	}
}

func TestDeriveRefusesWhatItCannotDerive(t *testing.T) {
	source := harbor(t)
	target := tenant(t, source)
	noVersion, noEnterprise := options(), options()
	noVersion.SourceVersion, noEnterprise.Enterprise = "", model.Ref{}
	person, runtime := source.OfKind(model.KindHumanIdentity)[0].ID, source.OfKind(model.KindAgentRuntime)[0].ID
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	for name, tc := range map[string]struct {
		ctx   context.Context
		dv    *Deriver
		roots []Root
		opts  Options
		want  error
	}{
		"record":            {roots: []Root{{ID: "PROCINST-OE-2026-0042"}}, opts: options(), want: ErrNotDerivable},
		"person":            {roots: []Root{{ID: person}}, opts: options(), want: ErrNotDerivable},
		"runtime":           {roots: []Root{{ID: runtime}}, opts: options(), want: ErrNotDerivable},
		"enterprise":        {roots: []Root{{ID: "ENT-HARBOR"}}, opts: options(), want: ErrNotDerivable},
		"no source version": {roots: []Root{{ID: "RESP-COORDINATE-ORDER-EXCEPTION"}}, opts: noVersion, want: ErrNoSourceVersion},
		"no enterprise":     {roots: []Root{{ID: "TASK-OE-DETECT"}}, opts: noEnterprise, want: ErrNoEnterprise},
		"unknown version":   {roots: []Root{{ID: "TASK-OE-DETECT", Version: "9"}}, opts: options(), want: corpus.ErrNotFound},
		"unknown root":      {roots: []Root{{ID: "NONE"}}, opts: options(), want: corpus.ErrNotFound},
		"no roots":          {opts: options(), want: ErrInvalidRequest},
		"blank root":        {roots: []Root{{}}, opts: options(), want: ErrInvalidRequest},
		"two versions":      {roots: []Root{{ID: "T", Version: "1"}, {ID: "T", Version: "2"}}, opts: options(), want: ErrInvalidRequest},
		"same namespace":    {roots: []Root{{ID: "T"}}, opts: Options{SourceNamespace: src, TargetNamespace: src}, want: ErrInvalidRequest},
		"no namespace":      {roots: []Root{{ID: "T"}}, opts: Options{SourceNamespace: src}, want: ErrInvalidRequest},
		"zero deriver":      {dv: &Deriver{}, roots: []Root{{ID: "T"}}, opts: options(), want: ErrInvalidRequest},
		"canceled":          {ctx: canceled, roots: []Root{{ID: "TASK-OE-DETECT"}}, opts: options(), want: context.Canceled},
	} {
		ctx, dv := tc.ctx, tc.dv
		if ctx == nil {
			ctx = context.Background()
		}
		if dv == nil {
			dv = deriver(t)
		}
		if out, err := dv.Derive(ctx, source, target, tc.roots, tc.opts); !errors.Is(err, tc.want) || out != nil {
			t.Errorf("%s: %d copies, %v", name, len(out), err)
		}
	}
}
