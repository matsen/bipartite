package flow

import (
	"os"
	"sync"
)

// File-level cache for sources.yml and the nexus config.yml. Both files are
// read repeatedly from a single command invocation (every flow.LoadSources
// helper, every flow.ResolveRepoPath call), and they never change mid-run.
// We key on (path, mtime, size) so a hot reload — say a test that rewrites
// the file in-place — invalidates automatically.
//
// The global ~/.config/bip/config.yml has its own cache in internal/config;
// this cache covers the two nexus-relative files.

type cacheEntry struct {
	mtimeNS int64
	size    int64
	value   interface{}
}

var (
	fileCacheMu sync.Mutex
	fileCache   = map[string]cacheEntry{}
)

// cachedLoad returns a cached parse if the file's mtime+size match a prior
// read, otherwise it invokes load, caches the result, and returns it. The
// load closure is responsible for reading the file itself — cachedLoad does
// not pass the bytes through, so the on-disk content has one canonical
// parsing pathway. Any error from load is returned without caching.
//
// Callers must use a stable absolute path (or at least the same string each
// time) — relative paths under different working directories will appear as
// distinct cache keys.
func cachedLoad(path string, load func() (interface{}, error)) (interface{}, error) {
	info, statErr := os.Stat(path)
	// If stat fails (missing file), skip the cache fast-path and let load()
	// produce the canonical error. We do not negatively cache.
	if statErr == nil {
		fileCacheMu.Lock()
		entry, ok := fileCache[path]
		fileCacheMu.Unlock()
		if ok && entry.mtimeNS == info.ModTime().UnixNano() && entry.size == info.Size() {
			return entry.value, nil
		}
	}
	val, err := load()
	if err != nil {
		return nil, err
	}
	// Re-stat after a successful read so the cache key reflects the file we
	// actually parsed (between the first stat and the read, the file could
	// have rotated).
	info, statErr = os.Stat(path)
	if statErr == nil {
		fileCacheMu.Lock()
		fileCache[path] = cacheEntry{
			mtimeNS: info.ModTime().UnixNano(),
			size:    info.Size(),
			value:   val,
		}
		fileCacheMu.Unlock()
	}
	return val, nil
}

// ResetFileCache clears the flow-package file cache. Intended for tests that
// rewrite sources.yml or config.yml in place at sub-second granularity (where
// mtime+size may not distinguish the two versions).
func ResetFileCache() {
	fileCacheMu.Lock()
	fileCache = map[string]cacheEntry{}
	fileCacheMu.Unlock()
}
