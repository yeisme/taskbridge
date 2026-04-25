package cmd

import (
	"github.com/yeisme/taskbridge/internal/storage"
	"github.com/yeisme/taskbridge/internal/storage/filestore"
)

// getStore creates a storage instance with lifecycle management.
// Returns the store and a cleanup function that flushes pending writes.
func getStore() (storage.Storage, func(), error) {
	fs, err := filestore.New(cfg.Storage.Path, cfg.Storage.File.Format)
	if err != nil {
		return nil, nil, err
	}

	var store storage.Storage = fs
	cleanup := func() {
		if f, ok := store.(storage.Flushable); ok {
			_ = f.Flush()
		}
	}

	return store, cleanup, nil
}
