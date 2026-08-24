package corpus

// Loader produces a corpus from some source.
type Loader interface {
	Load() (*Corpus, error)
}
