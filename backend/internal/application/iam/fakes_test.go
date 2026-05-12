package iam_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"sync"
	"time"

	domiam "github.com/micocards/api/internal/domain/iam"
)

type fakeUsers struct {
	byID    map[string]*domiam.User
	byEmail map[string]*domiam.User
	saveErr error
}

func newFakeUsers() *fakeUsers {
	return &fakeUsers{byID: map[string]*domiam.User{}, byEmail: map[string]*domiam.User{}}
}

func (f *fakeUsers) Save(_ context.Context, u *domiam.User) error {
	if f.saveErr != nil {
		return f.saveErr
	}
	if existing, ok := f.byEmail[u.Email().String()]; ok && existing.ID() != u.ID() {
		return domiam.ErrEmailTaken
	}
	if old, ok := f.byID[u.ID()]; ok {
		delete(f.byEmail, old.Email().String())
	}
	f.byID[u.ID()] = u
	f.byEmail[u.Email().String()] = u
	return nil
}

func (f *fakeUsers) ByID(_ context.Context, id string) (*domiam.User, error) {
	if u, ok := f.byID[id]; ok {
		return u, nil
	}
	return nil, domiam.ErrUserNotFound
}

func (f *fakeUsers) ByEmail(_ context.Context, e domiam.EmailAddress) (*domiam.User, error) {
	if u, ok := f.byEmail[e.String()]; ok {
		return u, nil
	}
	return nil, domiam.ErrUserNotFound
}

func (f *fakeUsers) EmailExists(_ context.Context, e domiam.EmailAddress) (bool, error) {
	_, ok := f.byEmail[e.String()]
	return ok, nil
}

type fakeRefreshTokens struct {
	byID   map[string]domiam.RefreshToken
	byHash map[string]string // hash -> id
	mu     sync.Mutex
}

func newFakeRefreshTokens() *fakeRefreshTokens {
	return &fakeRefreshTokens{
		byID:   map[string]domiam.RefreshToken{},
		byHash: map[string]string{},
	}
}

func (f *fakeRefreshTokens) Save(_ context.Context, t domiam.RefreshToken) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.byID[t.ID] = t
	f.byHash[t.OpaqueHash] = t.ID
	return nil
}

func (f *fakeRefreshTokens) ByOpaqueHash(_ context.Context, hash string) (domiam.RefreshToken, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.byHash[hash]
	if !ok {
		return domiam.RefreshToken{}, domiam.ErrRefreshTokenInvalid
	}
	return f.byID[id], nil
}

func (f *fakeRefreshTokens) FamilyByID(_ context.Context, familyID string) (domiam.RefreshTokenFamily, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fam := domiam.RefreshTokenFamily{FamilyID: familyID}
	for _, t := range f.byID {
		if t.FamilyID == familyID {
			fam.UserID = t.UserID
			fam.Tokens = append(fam.Tokens, t)
		}
	}
	return fam, nil
}

func (f *fakeRefreshTokens) RevokeFamily(_ context.Context, familyID string, _ domiam.TimeFn, reason domiam.RevokeReason) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now().UTC()
	for id, t := range f.byID {
		if t.FamilyID != familyID {
			continue
		}
		if t.RevokedAt != nil {
			continue
		}
		tt := now
		t.RevokedAt = &tt
		t.RevokeNote = reason
		f.byID[id] = t
	}
	return nil
}

func (f *fakeRefreshTokens) RevokeAllForUser(_ context.Context, userID string, _ domiam.TimeFn, reason domiam.RevokeReason) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	now := time.Now().UTC()
	for id, t := range f.byID {
		if t.UserID != userID {
			continue
		}
		if t.RevokedAt != nil {
			continue
		}
		tt := now
		t.RevokedAt = &tt
		t.RevokeNote = reason
		f.byID[id] = t
	}
	return nil
}

func (f *fakeRefreshTokens) RevokeOne(_ context.Context, tokenID string, _ domiam.TimeFn, reason domiam.RevokeReason) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.byID[tokenID]
	if !ok {
		return domiam.ErrRefreshTokenInvalid
	}
	if t.RevokedAt != nil {
		return nil
	}
	now := time.Now().UTC()
	t.RevokedAt = &now
	t.RevokeNote = reason
	f.byID[tokenID] = t
	return nil
}

type fakeHasher struct {
	weak       string
	hashSuffix int
	mu         sync.Mutex
}

const fakeBcryptPrefix = "$2a$10$"

func (h *fakeHasher) Hash(_ context.Context, plaintext string) (domiam.PasswordHash, error) {
	h.mu.Lock()
	h.hashSuffix++
	suffix := h.hashSuffix
	h.mu.Unlock()
	body := plaintext + "::" + strconv.Itoa(suffix)
	if len(body) < 53 {
		body = body + ":xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	}
	return domiam.NewPasswordHash(fakeBcryptPrefix + body[:53])
}

func (h *fakeHasher) Compare(_ context.Context, hash domiam.PasswordHash, plaintext string) error {
	if hash.IsZero() {
		return domiam.ErrInvalidCredentials
	}
	// fake-bcrypt format: prefix + plaintext::N + padding
	expectedPrefix := fakeBcryptPrefix + plaintext + "::"
	if len(hash.String()) < len(expectedPrefix) || hash.String()[:len(expectedPrefix)] != expectedPrefix {
		return domiam.ErrInvalidCredentials
	}
	return nil
}

func (h *fakeHasher) Strength(plaintext string) error {
	if plaintext == h.weak || len(plaintext) < 8 {
		return domiam.ErrPasswordTooWeak
	}
	return nil
}

type fakeAccessSigner struct {
	now func() time.Time
}

func (s *fakeAccessSigner) SignAccessToken(_ context.Context, userID string) (string, int64, error) {
	now := time.Now()
	if s.now != nil {
		now = s.now()
	}
	exp := now.Add(15 * time.Minute).Unix()
	return "access-" + userID + "-" + strconv.FormatInt(exp, 10), exp, nil
}

func (s *fakeAccessSigner) VerifyAccessToken(_ context.Context, token string) (string, error) {
	if token == "" {
		return "", domiam.ErrUnauthorized
	}
	return "u-from-token", nil
}

type fakeRefreshMinter struct {
	mu sync.Mutex
	n  int
}

func (m *fakeRefreshMinter) Mint(_ context.Context) (string, string, error) {
	m.mu.Lock()
	m.n++
	n := m.n
	m.mu.Unlock()
	plain := "refresh-plain-" + strconv.Itoa(n)
	return plain, sha256Hex(plain), nil
}

type fakeRefreshHasher struct{}

func (fakeRefreshHasher) HashOpaque(plaintext string) string { return sha256Hex(plaintext) }

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

type fakeUoW struct{ failures []error }

func (f *fakeUoW) Do(ctx context.Context, fn func(context.Context) error) error {
	if len(f.failures) > 0 {
		err := f.failures[0]
		f.failures = f.failures[1:]
		if err != nil {
			return err
		}
	}
	return fn(ctx)
}

type recordedEvent struct {
	Name    string
	Payload domiam.Event
}

type fakeIAMEvents struct {
	mu   sync.Mutex
	rows []recordedEvent
	err  error
}

func (e *fakeIAMEvents) Publish(_ context.Context, events ...domiam.Event) error {
	if e.err != nil {
		return e.err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, ev := range events {
		e.rows = append(e.rows, recordedEvent{Name: ev.Name(), Payload: ev})
	}
	return nil
}

func (e *fakeIAMEvents) Names() []string {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]string, len(e.rows))
	for i, r := range e.rows {
		out[i] = r.Name
	}
	return out
}

func (e *fakeIAMEvents) Reset() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rows = nil
	e.err = nil
}

var errBoom = errors.New("boom")
