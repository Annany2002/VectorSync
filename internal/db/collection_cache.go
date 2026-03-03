package db

import "sync"

// collectionInfo holds cached collection metadata
type collectionInfo struct {
	dimension      int32
	distanceMetric string
}

// CollectionCache stores collection metadata to avoid repeated DB lookups
type CollectionCache struct {
	mu    sync.RWMutex
	cache map[string]collectionInfo
}

// NewCollectionCache creates a new CollectionCache instance
func NewCollectionCache() *CollectionCache {
	return &CollectionCache{
		cache: make(map[string]collectionInfo),
	}
}

// CheckIdOrName checks if a collection exists in the cache by ID or name
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

// Insert adds a collection's metadata to the cache
func (c *CollectionCache) Insert(collectionId string, dimension int32, distanceMetric string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[collectionId] = collectionInfo{
		dimension:      dimension,
		distanceMetric: distanceMetric,
	}
}

// GetDimension returns the cached vector dimension for a collection ID.
// Returns (dimension, true) if found, (0, false) if not cached.
func (c *CollectionCache) GetDimension(collectionId string) (int32, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	info, exists := c.cache[collectionId]
	return info.dimension, exists
}

// GetDistanceMetric returns the cached distance metric for a collection ID.
// Returns (metric, true) if found, ("", false) if not cached.
func (c *CollectionCache) GetDistanceMetric(collectionId string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	info, exists := c.cache[collectionId]
	return info.distanceMetric, exists
}

// Delete removes a collection from the cache
func (c *CollectionCache) Delete(collectionId string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, collectionId)
}
