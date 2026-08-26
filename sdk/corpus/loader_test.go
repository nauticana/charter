package corpus

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestParserRequiresIDAndKind(t *testing.T) {
	if _, err := (Parser{}).Parse([]byte(`{"charterSpecVersion":"draft","namespace":"t","id":"X"}`)); err == nil {
		t.Error("document without kind accepted")
	}
	if _, err := (Parser{}).Parse([]byte(`not json`)); err == nil {
		t.Error("invalid JSON accepted")
	}
	d, err := (Parser{}).Parse([]byte(`{"charterSpecVersion":"draft","namespace":"t","id":"X","kind":"Enterprise","name":"x","extensions":{"vendor:x":{"n":1}}}`))
	if err != nil || d.Kind != "Enterprise" || len(d.Extensions) != 1 {
		t.Errorf("parse: %+v %v", d, err)
	}
}

func TestFSLoaderEnforcesFilenamesAndUniqueness(t *testing.T) {
	doc := func(id string) *fstest.MapFile {
		return &fstest.MapFile{Data: []byte(`{"charterSpecVersion":"draft","namespace":"t","id":"` + id + `","kind":"Enterprise","name":"x"}`)}
	}
	good := fstest.MapFS{"a/E1.json": doc("E1"), "b/E2.json": doc("E2"), "notes.md": &fstest.MapFile{Data: []byte("ignored")}}
	c, err := (&BaseFSLoader{FS: good, Root: "."}).Load()
	if err != nil || c.Len() != 2 {
		t.Fatalf("load: %d %v", c.Len(), err)
	}
	if _, err := (&BaseFSLoader{FS: fstest.MapFS{"E1.json": doc("OTHER")}, Root: "."}).Load(); err == nil || !strings.Contains(err.Error(), "does not match id") {
		t.Errorf("filename mismatch accepted: %v", err)
	}
	if _, err := (&BaseFSLoader{FS: fstest.MapFS{"a/E1.json": doc("E1"), "b/E1.json": doc("E1")}, Root: "."}).Load(); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("duplicate document accepted: %v", err)
	}
	if _, err := (&BaseFSLoader{FS: fstest.MapFS{"E1.json": &fstest.MapFile{Data: []byte("{")}}, Root: "."}).Load(); err == nil {
		t.Error("broken JSON accepted")
	}
}
