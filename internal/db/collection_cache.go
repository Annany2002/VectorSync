package db

import (
	"context"
	"database/sql"
	"sync"
)

// collectionInfo holds cached collection metadata
type collectionInfo struct {
	dimension         int32
	distanceMetric    string
	embeddingProvider string
	embeddingModel    string
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
func (c *CollectionCache) Insert(collectionId string, dimension int32, distanceMetric string, embeddingProvider, embeddingModel string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[collectionId] = collectionInfo{
		dimension:         dimension,
		distanceMetric:    distanceMetric,
		embeddingProvider: embeddingProvider,
		embeddingModel:    embeddingModel,
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

// GetInfo returns vector dimension, distance metric, embedding provider, and embedding model for a collection
// under a single read-lock acquisition, halving lock overhead on the hot path
// compared to calling helpers separately.
// Returns (dimension, metric, provider, model, true) if found, (0, "", "", "", false) if not cached.
func (c *CollectionCache) GetInfo(collectionId string) (int32, string, string, string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	info, exists := c.cache[collectionId]
	return info.dimension, info.distanceMetric, info.embeddingProvider, info.embeddingModel, exists
}

// Delete removes a collection from the cache
func (c *CollectionCache) Delete(collectionId string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.cache, collectionId)
}

// WarmFromDB loads all collections' dimension, distance_metric, embedding_provider, and embedding_model into the
// cache in a single query. Call this once at startup (in a background goroutine)
// to eliminate the first-request cache-miss DB round-trip for every collection.
func (c *CollectionCache) WarmFromDB(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, "SELECT id, vector_dim, distance_metric, embedding_provider, embedding_model FROM collections")
	if err != nil {
		return err
	}
	defer rows.Close()

	c.mu.Lock()
	defer c.mu.Unlock()

	for rows.Next() {
		var id string
		var dim int32
		var metric string
		var ep, em sql.NullString
		if err := rows.Scan(&id, &dim, &metric, &ep, &em); err != nil {
			return err
		}
		var epStr, emStr string
		if ep.Valid {
			epStr = ep.String
		}
		if em.Valid {
			emStr = em.String
		}
		c.cache[id] = collectionInfo{
			dimension:         dim,
			distanceMetric:    metric,
			embeddingProvider: epStr,
			embeddingModel:    emStr,
		}
	}
	return rows.Err()
}
