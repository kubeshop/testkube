package imageinspector

import (
	"context"
	"encoding/json"
	"os"
	"sync"
)

type fileStorage struct {
	filePath       string
	avoidDirectGet bool // if there is memory storage prior to this one, all the contents will be copied there anyway
	mu             sync.RWMutex
}

func NewFileStorage(filePath string, avoidDirectGet bool) StorageWithTransfer {
	return &fileStorage{
		filePath:       filePath,
		avoidDirectGet: avoidDirectGet,
	}
}

func (c *fileStorage) fetch(_ context.Context) (map[string]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	content, _ := os.ReadFile(c.filePath)
	cache := map[string]string{}
	if len(content) > 0 {
		_ = json.Unmarshal(content, &cache)
	}
	if cache == nil {
		cache = map[string]string{}
	}
	return cache, nil
}

func (c *fileStorage) save(ctx context.Context, serializedData map[string]string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// When the cache is too big, delete the oldest items and try again
	if len(serializedData) > 10000 { // todo: use const or configuration
		cleanOldRecords(&serializedData)
	}

	// Save data
	content, err := json.Marshal(serializedData)
	if err != nil {
		return err
	}
	err = os.WriteFile(c.filePath, content, 0644)

	return err
}

func (c *fileStorage) StoreMany(ctx context.Context, data map[Hash]Info) (err error) {
	if data == nil {
		return
	}
	serialized, err := c.fetch(ctx)
	if err != nil {
		return
	}
	for k, v := range data {
		serialized[string(k)], err = marshalInfo(v)
		if err != nil {
			return
		}
	}
	return c.save(ctx, serialized)
}

func (c *fileStorage) CopyTo(ctx context.Context, other ...StorageTransfer) (err error) {
	serialized, err := c.fetch(ctx)
	if err != nil {
		return
	}
	data := make(map[Hash]Info, len(serialized))
	for k, v := range serialized {
		data[Hash(k)], err = unmarshalInfo(v)
		if err != nil {
			return
		}
	}
	for _, v := range other {
		err = v.StoreMany(ctx, data)
		if err != nil {
			return
		}
	}
	return
}

func (c *fileStorage) Store(ctx context.Context, request RequestBase, info Info) error {
	return c.StoreMany(ctx, map[Hash]Info{
		hash(request.Registry, request.Image): info,
	})
}

func (c *fileStorage) Get(ctx context.Context, request RequestBase) (*Info, error) {
	if c.avoidDirectGet {
		return nil, nil
	}
	data, err := c.fetch(ctx)
	if err != nil {
		return nil, err
	}
	value, ok := data[string(hash(request.Registry, request.Image))]
	if !ok {
		return nil, nil
	}
	v, err := unmarshalInfo(value)
	if err != nil {
		return nil, err
	}
	return &v, nil
}
