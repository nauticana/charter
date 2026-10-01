package keel

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"time"

	kmodel "github.com/nauticana/keel/model"
	"github.com/nauticana/keel/port"
)

type docKey struct {
	partner       int64
	namespace, id string
}

type revisionRow struct {
	revision        int64
	version         any
	content, digest string
}

type membershipKey struct {
	partner, user int64
	begda         time.Time
}

type accountRow struct {
	partner          int64
	namespace, human string
	user             int64
	membershipBegda  time.Time
	begda            time.Time
	endda            *time.Time
}

type memoryState struct {
	kinds       map[docKey]string
	revisions   map[docKey][]revisionRow
	memberships map[membershipKey]*time.Time
	accounts    []accountRow
}

func (s memoryState) clone() memoryState {
	c := memoryState{kinds: maps.Clone(s.kinds), revisions: map[docKey][]revisionRow{}, memberships: maps.Clone(s.memberships), accounts: slices.Clone(s.accounts)}
	for k, v := range s.revisions {
		c.revisions[k] = slices.Clone(v)
	}
	return c
}

// memoryDB executes the adapter's named queries over in-memory tables; a transaction works on a copy that commit publishes.
type memoryDB struct {
	port.DatabaseRepository
	state      memoryState
	failQuery  string
	failCommit bool
	rollbacks  int
	calls      map[string]int
}

func newMemoryDB() *memoryDB {
	return &memoryDB{calls: map[string]int{}, state: memoryState{kinds: map[docKey]string{}, revisions: map[docKey][]revisionRow{}, memberships: map[membershipKey]*time.Time{}}}
}

func (db *memoryDB) GetQueryService(context.Context, map[string]string) port.QueryService {
	return &memoryQueries{db: db, state: &db.state}
}

func (db *memoryDB) BeginTx(context.Context, map[string]string) (port.TxQueryService, error) {
	state := db.state.clone()
	return &memoryTx{memoryQueries{db: db, state: &state}}, nil
}

type memoryQueries struct {
	db    *memoryDB
	state *memoryState
}

type memoryTx struct{ memoryQueries }

func (tx *memoryTx) Commit(context.Context) error {
	if tx.db.failCommit {
		return errors.New("commit failed")
	}
	tx.db.state = *tx.state
	return nil
}

func (tx *memoryTx) Rollback(context.Context) error {
	tx.db.rollbacks++
	return nil
}

func (q *memoryQueries) GenID() int64 { return 0 }

func rows(r ...[]any) *kmodel.QueryResult { return &kmodel.QueryResult{Rows: r} }

func (q *memoryQueries) Query(_ context.Context, name string, args ...any) (*kmodel.QueryResult, error) {
	q.db.calls[name]++
	if name == q.db.failQuery {
		return nil, fmt.Errorf("query %s failed", name)
	}
	s := q.state
	key := func(i int) docKey { return docKey{args[i].(int64), args[i+1].(string), args[i+2].(string)} }
	current := func(k docKey) (revisionRow, bool) {
		revs := s.revisions[k]
		if len(revs) == 0 {
			return revisionRow{}, false
		}
		return revs[len(revs)-1], true
	}
	sortedKeys := func(partner int64, kind string) []docKey {
		var out []docKey
		for k, v := range s.kinds {
			if k.partner == partner && v == kind {
				out = append(out, k)
			}
		}
		slices.SortFunc(out, func(a, b docKey) int {
			if a.namespace != b.namespace {
				return cmp.Compare(a.namespace, b.namespace)
			}
			return cmp.Compare(a.id, b.id)
		})
		return out
	}
	switch name {
	case qHumanDocumentKind:
		if kind, ok := s.kinds[key(0)]; ok {
			return rows([]any{kind}), nil
		}
		return rows(), nil
	case qDocumentClaim:
		k := key(0)
		kind, ok := s.kinds[k]
		if !ok {
			kind = args[3].(string)
			s.kinds[k] = kind
		}
		return rows([]any{kind}), nil
	case qRevisions:
		out := rows()
		revs := s.revisions[key(0)]
		for i := len(revs) - 1; i >= 0; i-- {
			out.Rows = append(out.Rows, []any{revs[i].revision, revs[i].version, revs[i].digest, revs[i].content})
		}
		return out, nil
	case qRevisionInsert:
		k := key(0)
		for _, r := range s.revisions[k] {
			if r.revision == args[3].(int64) || (args[4] != nil && r.version == args[4]) {
				return nil, errors.New("duplicate key charter_document_revision")
			}
		}
		s.revisions[k] = append(s.revisions[k], revisionRow{revision: args[3].(int64), version: args[4], content: args[5].(string), digest: args[6].(string)})
	case qDocumentCurrent:
		if r, ok := current(key(0)); ok {
			return rows([]any{r.content, r.digest}), nil
		}
		return rows(), nil
	case qDocumentsOfKind:
		out := rows()
		for _, k := range sortedKeys(args[0].(int64), args[1].(string)) {
			if r, ok := current(k); ok {
				out.Rows = append(out.Rows, []any{r.content, r.digest})
			}
		}
		return out, nil
	case qMembership:
		if endda, ok := s.memberships[membershipKey{args[0].(int64), args[1].(int64), args[2].(time.Time)}]; ok {
			if endda == nil {
				return rows([]any{nil}), nil
			}
			return rows([]any{*endda}), nil
		}
		return rows(), nil
	case qAccountOverlaps:
		out := rows()
		open := args[4].(bool)
		endda, finite := args[5].(time.Time)
		for _, a := range s.accounts {
			if a.partner == args[0] && ((a.namespace == args[1] && a.human == args[2]) || a.user == args[3]) &&
				(open || finite && a.begda.Before(endda)) && (a.endda == nil || a.endda.After(args[6].(time.Time))) {
				out.Rows = append(out.Rows, []any{a.namespace, a.human, a.user, a.begda, timeOrNil(a.endda)})
			}
		}
		return out, nil
	case qAccountInsert:
		a := accountRow{partner: args[0].(int64), namespace: args[1].(string), human: args[2].(string), user: args[3].(int64), membershipBegda: args[4].(time.Time), begda: args[5].(time.Time)}
		if args[6] != nil {
			endda := args[6].(time.Time)
			a.endda = &endda
		}
		s.accounts = append(s.accounts, a)
	case qAccountOpenAt:
		out := rows()
		for _, a := range s.accounts {
			if a.partner == args[0] && a.namespace == args[1] && a.human == args[2] && a.begda.Before(args[3].(time.Time)) && (a.endda == nil || a.endda.After(args[4].(time.Time))) {
				out.Rows = append(out.Rows, []any{a.begda})
			}
		}
		return out, nil
	case qAccountEnd:
		for i, a := range s.accounts {
			if a.partner == args[1] && a.namespace == args[2] && a.human == args[3] && a.begda.Equal(args[4].(time.Time)) {
				endda := args[0].(time.Time)
				s.accounts[i].endda = &endda
			}
		}
	case qHumanOfUser, qUserOfHuman:
		at := args[len(args)-1].(time.Time)
		out := rows()
		for _, a := range s.accounts {
			mine := a.user == args[1]
			if name == qUserOfHuman {
				mine = a.namespace == args[1] && a.human == args[2]
			}
			membershipEnd := s.memberships[membershipKey{a.partner, a.user, a.membershipBegda}]
			if a.partner == args[0] && mine && !a.begda.After(at) && (a.endda == nil || a.endda.After(at)) && (membershipEnd == nil || membershipEnd.After(at)) {
				if name == qHumanOfUser {
					out.Rows = append(out.Rows, []any{a.namespace, a.human})
				} else {
					out.Rows = append(out.Rows, []any{a.user})
				}
			}
		}
		return out, nil
	default:
		return nil, fmt.Errorf("unknown query %s", name)
	}
	return rows(), nil
}

func timeOrNil(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}
