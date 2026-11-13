package crud

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"hotwallet/backend/internal/infrastructure/store"
)

// Entity adalah kontrak minimum model yang disimpan di koleksi.
type Entity interface {
	GetID() string
}

// Sentinel error dipakai service dan dipetakan ke HTTP status oleh response.
var (
	ErrNotFound     = errors.New("not found")
	ErrInvalid      = errors.New("invalid input")
	ErrConflict     = errors.New("conflict")
	ErrInternal     = errors.New("internal error")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
)

// Is re-export errors.Is for convenience.
var Is = errors.Is

// Invalid membungkus ErrInvalid dengan detail yang bisa dibaca.
func Invalid(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}

// NotFoundf membungkus ErrNotFound dengan detail yang bisa dibaca.
func NotFoundf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrNotFound, fmt.Sprintf(format, args...))
}

// Conflictf membungkus ErrConflict dengan detail yang bisa dibaca.
func Conflictf(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrConflict, fmt.Sprintf(format, args...))
}

// Repo adalah koleksi generik berbasis SQLite.
// T harus implement GetID() dan SetID(string).
type Repo[T Entity] struct {
	store      *store.Store
	collection string
	mu         sync.Mutex
}

func NewRepo[T Entity](s *store.Store, collection string) *Repo[T] {
	return &Repo[T]{store: s, collection: collection}
}

func (r *Repo[T]) All() ([]T, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.load()
}

// GetAll adalah alias dari All, mengikuti penamaan baku di struktur target.
func (r *Repo[T]) GetAll() ([]T, error) {
	return r.All()
}

func (r *Repo[T]) Get(id string) (T, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	items, err := r.load()
	if err != nil {
		var zero T
		return zero, false, err
	}
	for _, item := range items {
		if item.GetID() == id {
			return item, true, nil
		}
	}
	var zero T
	return zero, false, nil
}

// GetByID adalah alias dari Get, mengikuti penamaan baku di struktur target.
func (r *Repo[T]) GetByID(id string) (T, bool, error) {
	return r.Get(id)
}

func (r *Repo[T]) Create(item T) (T, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var zero T
	if item.GetID() == "" {
		if err := setID(&item, newID()); err != nil {
			return zero, err
		}
	}

	items, err := r.load()
	if err != nil {
		return zero, err
	}
	items = append(items, item)
	if err := r.store.Save(r.collection, items); err != nil {
		return zero, err
	}
	return item, nil
}

func (r *Repo[T]) Update(id string, item T) (T, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var zero T
	items, err := r.load()
	if err != nil {
		return zero, false, err
	}
	for i := range items {
		if items[i].GetID() != id {
			continue
		}
		if err := setID(&item, id); err != nil {
			return zero, false, err
		}
		items[i] = item
		if err := r.store.Save(r.collection, items); err != nil {
			return zero, false, err
		}
		return item, true, nil
	}
	return zero, false, nil
}

// Patch merge partial JSON ke item yang sudah ada.
func (r *Repo[T]) Patch(id string, patch []byte) (T, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var zero T
	items, err := r.load()
	if err != nil {
		return zero, false, err
	}
	for i := range items {
		if items[i].GetID() != id {
			continue
		}
		merged, err := mergeJSON(items[i], patch)
		if err != nil {
			return zero, false, err
		}
		items[i] = merged
		if err := r.store.Save(r.collection, items); err != nil {
			return zero, false, err
		}
		return merged, true, nil
	}
	return zero, false, nil
}

func (r *Repo[T]) Delete(id string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	items, err := r.load()
	if err != nil {
		return false, err
	}
	for i := range items {
		if items[i].GetID() != id {
			continue
		}
		items = append(items[:i], items[i+1:]...)
		if err := r.store.Save(r.collection, items); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (r *Repo[T]) load() ([]T, error) {
	var items []T
	if err := r.store.Load(r.collection, &items); err != nil {
		return nil, err
	}
	if items == nil {
		items = []T{}
	}
	return items, nil
}

func setID[T any](item *T, id string) error {
	if setter, ok := any(item).(interface{ SetID(string) }); ok {
		setter.SetID(id)
		return nil
	}
	return errors.New("model tidak implement SetID(string)")
}

func newID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "generated-id"
	}
	return hex.EncodeToString(b)
}

func mergeJSON[T any](base T, patch []byte) (T, error) {
	var zero T

	baseJSON, err := json.Marshal(base)
	if err != nil {
		return zero, err
	}

	var baseFields map[string]json.RawMessage
	if err := json.Unmarshal(baseJSON, &baseFields); err != nil {
		return zero, err
	}

	var patchFields map[string]json.RawMessage
	if err := json.Unmarshal(patch, &patchFields); err != nil {
		return zero, fmt.Errorf("body harus JSON object")
	}

	for key, value := range patchFields {
		baseFields[key] = value
	}

	mergedJSON, err := json.Marshal(baseFields)
	if err != nil {
		return zero, err
	}

	var merged T
	if err := json.Unmarshal(mergedJSON, &merged); err != nil {
		return zero, err
	}
	return merged, nil
}
