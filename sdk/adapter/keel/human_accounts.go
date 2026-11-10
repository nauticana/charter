package keel

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nauticana/keel/common"
	"github.com/nauticana/keel/port"

	"github.com/nauticana/charter/sdk/model"
)

var (
	ErrInvalidAccountLink = errors.New("invalid human account link")
	ErrNoMembership       = errors.New("no keel partner membership covers the link")
	ErrNotHumanIdentity   = errors.New("link target is not a stored HumanIdentity")
	ErrAccountLinkOverlap = errors.New("human account link overlaps another link of the human or user")
)

// HumanAccount links a keel partner membership (partner_user) to the Charter HumanIdentity it acts as, for the
// half-open period [Begda, Endda).
type HumanAccount struct {
	PartnerID        int64
	Human            model.Ref
	UserID           int64
	PartnerUserBegda time.Time
	Begda            time.Time
	Endda            *time.Time
}

const (
	qMembership      = "charter.human_account.membership"
	qHumanObject     = "charter.human_account.human"
	qAccountOverlaps = "charter.human_account.overlaps"
	qAccountInsert   = "charter.human_account.insert"
	qAccountOpenAt   = "charter.human_account.open_at"
	qAccountEnd      = "charter.human_account.end"
	qHumanOfUser     = "charter.human_account.human_of_user"
	qUserOfHuman     = "charter.human_account.user_of_human"
	effectiveAccount = `a.begda <= ? AND (a.endda IS NULL OR a.endda > ?) AND (p.endda IS NULL OR p.endda > ?)`
	membershipJoin   = `JOIN partner_user p ON p.partner_id = a.partner_id AND p.user_id = a.user_id AND p.begda = a.partner_user_begda`
	humanJoin        = `JOIN enterprise_object o ON o.partner_id = a.partner_id AND o.id = a.human_identity_id`
)

var humanKindCode = kindCodes.codes[string(model.KindHumanIdentity)]

var humanAccountQueries = map[string]string{
	qMembership: `
SELECT endda FROM partner_user WHERE partner_id = ? AND user_id = ? AND begda = ? FOR UPDATE`,
	qHumanObject: `
SELECT id FROM enterprise_object WHERE partner_id = ? AND namespace = ? AND identifier = ? AND kind = '` + humanKindCode + `'`,
	qAccountOverlaps: `
SELECT a.human_identity_id, a.user_id, a.begda, a.endda FROM human_identity_partner_user a
 WHERE a.partner_id = ? AND (a.human_identity_id = ? OR a.user_id = ?)
   AND (? OR a.begda < ?) AND (a.endda IS NULL OR a.endda > ?)`,
	qAccountInsert: `
INSERT INTO human_identity_partner_user (partner_id, human_identity_id, begda, user_id, partner_user_begda, endda)
VALUES (?, ?, ?, ?, ?, ?)`,
	qAccountOpenAt: `
SELECT a.begda FROM human_identity_partner_user a
 WHERE a.partner_id = ? AND a.human_identity_id = ? AND a.begda < ? AND (a.endda IS NULL OR a.endda > ?) FOR UPDATE`,
	qAccountEnd: `
UPDATE human_identity_partner_user SET endda = ? WHERE partner_id = ? AND human_identity_id = ? AND begda = ?`,
	qHumanOfUser: `
SELECT o.namespace, o.identifier FROM human_identity_partner_user a ` + membershipJoin + ` ` + humanJoin + `
 WHERE a.partner_id = ? AND a.user_id = ? AND ` + effectiveAccount,
	qUserOfHuman: `
SELECT a.user_id FROM human_identity_partner_user a ` + membershipJoin + ` ` + humanJoin + `
 WHERE a.partner_id = ? AND o.namespace = ? AND o.identifier = ? AND o.kind = '` + humanKindCode + `' AND ` + effectiveAccount,
}

// HumanAccounts keeps human_identity_partner_user links. A link lies within its membership, targets a stored
// HumanIdentity, and never overlaps another link of the same human or the same user, so each resolves to one
// counterpart at any time; a link whose membership has ended no longer resolves.
type HumanAccounts struct {
	db      port.DatabaseRepository
	queries port.QueryService
	now     func() time.Time
}

// NewHumanAccounts composes the store; a nil now uses the system clock for links that name no begda.
func NewHumanAccounts(ctx context.Context, db port.DatabaseRepository, now func() time.Time) (*HumanAccounts, error) {
	if db == nil {
		return nil, ErrNoDatabase
	}
	queries := db.GetQueryService(ctx, humanAccountQueries)
	if queries == nil {
		return nil, fmt.Errorf("%w: no query service", ErrNoDatabase)
	}
	if now == nil {
		now = time.Now
	}
	return &HumanAccounts{db: db, queries: queries, now: now}, nil
}

// Link records a new link; replaying an identical link changes nothing.
func (a *HumanAccounts) Link(ctx context.Context, link HumanAccount) (err error) {
	if link.Begda.IsZero() {
		link.Begda = a.now()
	}
	link = link.stored()
	if err := link.validate(); err != nil {
		return err
	}
	tx, err := a.db.BeginTx(ctx, humanAccountQueries)
	if err != nil {
		return fmt.Errorf("human account: begin: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			err = errors.Join(err, tx.Rollback(context.WithoutCancel(ctx)))
		}
	}()
	membership, err := tx.Query(ctx, qMembership, link.PartnerID, link.UserID, link.PartnerUserBegda)
	if err != nil {
		return fmt.Errorf("human account membership: %w", err)
	}
	if len(membership.Rows) == 0 {
		return fmt.Errorf("%w: user %d in partner %d since %s", ErrNoMembership, link.UserID, link.PartnerID, link.PartnerUserBegda.Format(time.RFC3339))
	}
	if ends, ended := common.AsTimeOK(membership.Rows[0][0]); ended && (link.Endda == nil || link.Endda.After(ends)) {
		return fmt.Errorf("%w: membership ends %s", ErrNoMembership, ends.Format(time.RFC3339))
	}
	human, err := humanObject(ctx, tx, link.PartnerID, link.Human)
	if err != nil {
		return err
	}
	var endda any
	if link.Endda != nil {
		endda = *link.Endda
	}
	overlaps, err := tx.Query(ctx, qAccountOverlaps, link.PartnerID, human, link.UserID, link.Endda == nil, endda, link.Begda)
	if err != nil {
		return fmt.Errorf("human account overlaps: %w", err)
	}
	for _, row := range overlaps.Rows {
		if !link.sameAs(human, row) {
			return fmt.Errorf("%w: %s:%s or user %d is already linked from %s", ErrAccountLinkOverlap, link.Human.Namespace, link.Human.ID, link.UserID,
				common.AsTime(row[2]).Format(time.RFC3339))
		}
	}
	if len(overlaps.Rows) == 0 {
		if _, err := tx.Query(ctx, qAccountInsert, link.PartnerID, human, link.Begda, link.UserID, link.PartnerUserBegda, endda); err != nil {
			return fmt.Errorf("human account insert: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("human account: commit: %w", err)
	}
	committed = true
	return nil
}

// End closes the human's link effective at endda; a human with no such link is an error.
func (a *HumanAccounts) End(ctx context.Context, partnerID int64, human model.Ref, endda time.Time) (err error) {
	if partnerID <= 0 || human.ID == "" || endda.IsZero() {
		return fmt.Errorf("%w: partner, human, and end are required", ErrInvalidAccountLink)
	}
	endda = storedTime(endda)
	tx, err := a.db.BeginTx(ctx, humanAccountQueries)
	if err != nil {
		return fmt.Errorf("human account: begin: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			err = errors.Join(err, tx.Rollback(context.WithoutCancel(ctx)))
		}
	}()
	id, err := humanObject(ctx, tx, partnerID, human)
	if err != nil {
		return err
	}
	open, err := tx.Query(ctx, qAccountOpenAt, partnerID, id, endda, endda)
	if err != nil {
		return fmt.Errorf("human account: %w", err)
	}
	if len(open.Rows) != 1 {
		return fmt.Errorf("%w: %s:%s has no link open at %s", ErrUnmapped, human.Namespace, human.ID, endda.Format(time.RFC3339))
	}
	if _, err := tx.Query(ctx, qAccountEnd, endda, partnerID, id, open.Rows[0][0]); err != nil {
		return fmt.Errorf("human account end: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("human account: commit: %w", err)
	}
	committed = true
	return nil
}

// Human returns the HumanIdentity the user acts as in the partner at the given time.
func (a *HumanAccounts) Human(ctx context.Context, partnerID, userID int64, at time.Time) (model.Ref, error) {
	if partnerID <= 0 || userID <= 0 {
		return model.Ref{}, fmt.Errorf("%w: partner %d user %d", ErrUnmapped, partnerID, userID)
	}
	result, err := a.queries.Query(ctx, qHumanOfUser, partnerID, userID, at, at, at)
	if err != nil {
		return model.Ref{}, fmt.Errorf("human account: %w", err)
	}
	if len(result.Rows) != 1 {
		return model.Ref{}, fmt.Errorf("%w: user %d has %d human links in partner %d", ErrUnmapped, userID, len(result.Rows), partnerID)
	}
	return model.Ref{Namespace: common.AsString(result.Rows[0][0]), ID: common.AsString(result.Rows[0][1])}, nil
}

// User returns the keel user acting as the human in the partner at the given time.
func (a *HumanAccounts) User(ctx context.Context, partnerID int64, human model.Ref, at time.Time) (int64, error) {
	if partnerID <= 0 || human.ID == "" {
		return 0, fmt.Errorf("%w: partner %d human %q", ErrUnmapped, partnerID, human.ID)
	}
	result, err := a.queries.Query(ctx, qUserOfHuman, partnerID, human.Namespace, human.ID, at, at, at)
	if err != nil {
		return 0, fmt.Errorf("human account: %w", err)
	}
	if len(result.Rows) != 1 {
		return 0, fmt.Errorf("%w: %s:%s has %d account links in partner %d", ErrUnmapped, human.Namespace, human.ID, len(result.Rows), partnerID)
	}
	userID, ok := common.AsInt64OK(result.Rows[0][0])
	if !ok || userID <= 0 {
		return 0, fmt.Errorf("%w: invalid user id for %s:%s", ErrUnmapped, human.Namespace, human.ID)
	}
	return userID, nil
}

func (l HumanAccount) validate() error {
	switch {
	case l.PartnerID <= 0 || l.UserID <= 0:
		return fmt.Errorf("%w: partner and user are required", ErrInvalidAccountLink)
	case l.Human.ID == "":
		return fmt.Errorf("%w: human is required", ErrInvalidAccountLink)
	case l.PartnerUserBegda.IsZero():
		return fmt.Errorf("%w: the membership begda is required", ErrInvalidAccountLink)
	case l.Begda.Before(l.PartnerUserBegda):
		return fmt.Errorf("%w: link begins before its membership", ErrInvalidAccountLink)
	case l.Endda != nil && !l.Endda.After(l.Begda):
		return fmt.Errorf("%w: endda must follow begda", ErrInvalidAccountLink)
	}
	return nil
}

// stored normalizes times to the UTC microseconds a TIMESTAMP column keeps, so a replay compares equal.
func (l HumanAccount) stored() HumanAccount {
	l.PartnerUserBegda = storedTime(l.PartnerUserBegda)
	l.Begda = storedTime(l.Begda)
	if l.Endda != nil {
		endda := storedTime(*l.Endda)
		l.Endda = &endda
	}
	return l
}

func storedTime(t time.Time) time.Time { return t.UTC().Truncate(time.Microsecond) }

// sameAs reports whether an overlapping row is this exact link, which makes Link a replay.
func (l HumanAccount) sameAs(human int64, row []any) bool {
	begda, _ := common.AsTimeOK(row[2])
	endda, open := common.AsTimeOK(row[3])
	sameEnd := (l.Endda == nil && !open) || (l.Endda != nil && open && l.Endda.Equal(endda))
	return common.AsInt64(row[0]) == human && common.AsInt64(row[1]) == l.UserID && begda.Equal(l.Begda) && sameEnd
}

// humanObject resolves a stored HumanIdentity to its object id.
func humanObject(ctx context.Context, q port.QueryService, partnerID int64, human model.Ref) (int64, error) {
	res, err := q.Query(ctx, qHumanObject, partnerID, human.Namespace, human.ID)
	if err != nil {
		return 0, fmt.Errorf("human account identity: %w", err)
	}
	if len(res.Rows) != 1 {
		return 0, fmt.Errorf("%w: %s:%s", ErrNotHumanIdentity, human.Namespace, human.ID)
	}
	id, ok := common.AsInt64OK(res.Rows[0][0])
	if !ok {
		return 0, fmt.Errorf("%w: %s:%s", ErrNotHumanIdentity, human.Namespace, human.ID)
	}
	return id, nil
}
