package evidence

import (
	"context"
	"fmt"
	"sync"

	"github.com/nauticana/charter/sdk/corpus"
	"github.com/nauticana/charter/sdk/model"
)

// BaseMemoryStore keeps evidence in an in-memory corpus.
type BaseMemoryStore struct {
	mu     sync.RWMutex
	corpus *corpus.Corpus
}

var _ Store = (*BaseMemoryStore)(nil)

func NewBaseMemoryStore() *BaseMemoryStore {
	return &BaseMemoryStore{corpus: corpus.New()}
}

func (s *BaseMemoryStore) Append(_ context.Context, d *model.Document) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.corpus.Get(d.Namespace, d.ID); exists {
		return fmt.Errorf("%w: %s:%s", ErrDuplicate, d.Namespace, d.ID)
	}
	return s.corpus.Add(d)
}

func (s *BaseMemoryStore) Fetch(ctx context.Context, key corpus.DocumentKey) (*model.Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.corpus.Fetch(ctx, key)
}

func (s *BaseMemoryStore) List(ctx context.Context, kind model.Kind) ([]*model.Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.corpus.List(ctx, kind)
}
