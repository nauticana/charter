package corpus

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// BaseFSLoader reads every *.json file below Root of FS; filenames must equal the document id.
type BaseFSLoader struct {
	FS     fs.FS
	Root   string
	Parser Parser
}

var _ Loader = (*BaseFSLoader)(nil)

func NewDirLoader(dir string) *BaseFSLoader {
	return &BaseFSLoader{FS: os.DirFS(dir), Root: "."}
}

func (l *BaseFSLoader) Load() (*Corpus, error) {
	c := New()
	err := fs.WalkDir(l.FS, l.Root, func(path string, e fs.DirEntry, err error) error {
		if err != nil || e.IsDir() || !strings.HasSuffix(path, ".json") {
			return err
		}
		b, err := fs.ReadFile(l.FS, path)
		if err != nil {
			return err
		}
		d, err := l.Parser.Parse(b)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if filepath.Base(path) != d.ID+".json" {
			return fmt.Errorf("%s: filename does not match id %s", path, d.ID)
		}
		return c.Add(d)
	})
	return c, err
}
