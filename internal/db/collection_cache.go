package db

import "sync"

// Collection cache will store the dimension of already existing collections avoiding repeated db lookups
type CollectionCache struct {
	mu    sync.RWMutex
	cache map[string]int32
}

// NewCollectionCache will create a new instance of ColletionCache
func NewCollectionCache() *CollectionCache {
	return &CollectionCache{
		cache: make(map[string]int32),
	}
}

// Check if collection id or name exists (either one of these have to be present)
func (c *CollectionCache) CheckIdOrName(collectionId, name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var res string

	if name == "" {
		res = collectionId
	} else {
		res = name
	}

	_, exists := c.cache[res]
	return exists
}

// Insert an collection id
func (c *CollectionCache) Insert(collectionId string, dimension int32) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[collectionId] = dimension
}

// GetDimension returns the cached vector dimension for a collection ID.
// Returns (dimension, true) if found, (0, false) if not cached.
func (c *CollectionCache) GetDimension(collectionId string) (int32, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	dim, exists := c.cache[collectionId]
	return dim, exists
}

// Delete removes a collection from the cache
func (c *CollectionCache) Delete(collectionId string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, collectionId)
}
