package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/nauticana/charter/sdk/interchange"
)

func exportCmd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("export", flag.ContinueOnError)
	fs.SetOutput(stderr)
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(stderr, "charter export: at least one document directory is required")
		return exitUsage
	}
	codec, err := interchange.New()
	if err != nil {
		return fail(stderr, err)
	}
	c, err := loadDirs(fs.Args())
	if err != nil {
		return fail(stderr, err)
	}
	if err := writeJSON(stdout, codec.Export(c.Documents())); err != nil {
		return fail(stderr, err)
	}
	return exitOK
}

// importCmd writes each document to <out>/<escaped namespace>/<id>.json; the imported set is then validated like any other.
func importCmd(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("import", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("out", "", "directory the documents are written into (required)")
	accept := fs.String("accept-loss", "", "comma-separated loss entries, as a refused import lists them, the caller accepts")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() != 1 || *out == "" {
		fmt.Fprintln(stderr, "charter import: one JSON-LD file and -out are required")
		return exitUsage
	}
	codec, err := interchange.New()
	if err != nil {
		return fail(stderr, err)
	}
	b, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		return fail(stderr, err)
	}
	var graph map[string]any
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	if err := dec.Decode(&graph); err != nil {
		return fail(stderr, err)
	}
	if dec.Decode(new(any)) != io.EOF {
		return fail(stderr, fmt.Errorf("%s holds more than one JSON value", fs.Arg(0)))
	}
	var accepted []string
	if *accept != "" {
		accepted = strings.Split(*accept, ",")
	}
	docs, loss, err := codec.Import(graph, accepted)
	if err != nil {
		fmt.Fprintf(stderr, "charter import: %v\n", err)
		return exitFailed
	}
	for _, d := range docs {
		// The namespace is a URI or token, so it is escaped; the id pattern admits no separator. Symbolic links are never followed.
		dir := url.QueryEscape(d.Namespace)
		if !filepath.IsLocal(dir) || !filepath.IsLocal(d.ID) {
			return fail(stderr, fmt.Errorf("document %s:%s is unsafe as a file path", d.Namespace, d.ID))
		}
		path := filepath.Join(*out, dir, d.ID+".json")
		for _, p := range []string{filepath.Dir(path), path} {
			if info, err := os.Lstat(p); err == nil && info.Mode()&os.ModeSymlink != 0 {
				return fail(stderr, fmt.Errorf("%s is a symbolic link", p))
			}
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fail(stderr, err)
		}
		var pretty bytes.Buffer
		if err := json.Indent(&pretty, d.Raw, "", "  "); err != nil {
			return fail(stderr, err)
		}
		if err := os.WriteFile(path, append(pretty.Bytes(), '\n'), 0o644); err != nil {
			return fail(stderr, err)
		}
	}
	fmt.Fprintf(stdout, "imported %d documents into %s", len(docs), *out)
	if len(loss) > 0 {
		fmt.Fprintf(stdout, ", accepting loss of %s", strings.Join(loss, "; "))
	}
	fmt.Fprintln(stdout)
	return exitOK
}

func contextCmd(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 {
		fmt.Fprintln(stderr, "charter context takes no arguments")
		return exitUsage
	}
	codec, err := interchange.New()
	if err != nil {
		return fail(stderr, err)
	}
	if err := writeJSON(stdout, codec.Context()); err != nil {
		return fail(stderr, err)
	}
	return exitOK
}
