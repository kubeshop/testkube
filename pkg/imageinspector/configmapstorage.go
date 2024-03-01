package imageinspector

import (
	"context"
	"encoding/json"
	"slices"
	"sync"
	"time"

	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/kubeshop/testkube/pkg/configmap"
)

type configmapStorage struct {
	client         configmap.Interface
	name           string
	avoidDirectGet bool // if there is memory storage prior to this one, all the contents will be copied there anyway
	mu             sync.RWMutex
}

func NewConfigMapStorage(client configmap.Interface, name string, avoidDirectGet bool) StorageWithTransfer {
	return &configmapStorage{
		client:         client,
		name:           name,
		avoidDirectGet: avoidDirectGet,
	}
}

func (c *configmapStorage) fetch(ctx context.Context) (map[string]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cache, err := c.client.Get(ctx, c.name)
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, errors.Wrap(err, "getting configmap cache")
	}
	if cache == nil {
		cache = map[string]string{}
	}
	return cache, nil
}

func cleanOldRecords(currentData *map[string]string) {
	// Unmarshal the fetched date for the records
	type Entry struct {
		time time.Time
		name string
	}
	dates := make([]Entry, 0, len(*currentData))
	var vv Info
	for k := range *currentData {
		_ = json.Unmarshal([]byte((*currentData)[k]), &vv)
		dates = append(dates, Entry{time: vv.FetchedAt, name: k})
	}
	slices.SortFunc(dates, func(a, b Entry) int {
		if a.time.Before(b.time) {
			return -1
		}
		return 1
	})

	// Delete half of the records
	for i := 0; i < len(*currentData)/2; i++ {
		delete(*currentData, dates[i].name)
	}
}

func (c *configmapStorage) save(ctx context.Context, serializedData map[string]string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Save data
	err := c.client.Apply(ctx, c.name, serializedData)

	// When the cache is too big, delete the oldest items and try again
	if err != nil && k8serrors.IsRequestEntityTooLargeError(err) {
		cleanOldRecords(&serializedData)
		err = c.client.Apply(ctx, c.name, serializedData)
	}
	return err
}

func (c *configmapStorage) StoreMany(ctx context.Context, data map[Hash]Info) (err error) {
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

func (c *configmapStorage) CopyTo(ctx context.Context, other ...StorageTransfer) (err error) {
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

func (c *configmapStorage) Store(ctx context.Context, request RequestBase, info Info) error {
	return c.StoreMany(ctx, map[Hash]Info{
		hash(request.Registry, request.Image): info,
	})
}

func (c *configmapStorage) Get(ctx context.Context, request RequestBase) (*Info, error) {
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
