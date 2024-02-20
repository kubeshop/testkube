package imageinspector

import (
	"context"
	"sync"
)

type memoryStorage struct {
	data map[Hash]Info
	mu   sync.RWMutex
}

func NewMemoryStorage() StorageWithTransfer {
	return &memoryStorage{
		data: make(map[Hash]Info),
	}
}

func (m *memoryStorage) StoreMany(_ context.Context, data map[Hash]Info) error {
	if data == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, v := range data {
		if vv, ok := m.data[k]; !ok || v.FetchedAt.After(vv.FetchedAt) {
			m.data[k] = v
		}
	}
	return nil
}

func (m *memoryStorage) CopyTo(ctx context.Context, other ...StorageTransfer) (err error) {
	if len(other) == 0 {
		return
	}
	for _, v := range other {
		err = v.StoreMany(ctx, m.data)
		if err != nil {
			return
		}
	}
	return
}

func (m *memoryStorage) Store(ctx context.Context, request RequestBase, info Info) error {
	return m.StoreMany(ctx, map[Hash]Info{
		hash(request.Registry, request.Image): info,
	})
}

func (m *memoryStorage) Get(_ context.Context, request RequestBase) (*Info, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if v, ok := m.data[hash(request.Registry, request.Image)]; ok {
		return &v, nil
	}
	return nil, nil
}
