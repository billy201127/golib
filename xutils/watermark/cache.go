package watermark

import (
	"container/list"
	"sync"
)

// watermarkLRU is a simple thread-safe LRU cache for watermark PNG bytes.
// Key: text + alpha + fontSize (quantized)，Value: PNG bytes.
type watermarkLRU struct {
	capacity int
	mu       sync.Mutex
	ll       *list.List
	cache    map[string]*list.Element
}

type watermarkEntry struct {
	key  string
	data []byte
}

func newWatermarkLRU(cap int) *watermarkLRU {
	if cap <= 0 {
		cap = 64
	}
	return &watermarkLRU{
		capacity: cap,
		ll:       list.New(),
		cache:    make(map[string]*list.Element, cap),
	}
}

func (l *watermarkLRU) Get(key string) ([]byte, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if ele, ok := l.cache[key]; ok {
		l.ll.MoveToFront(ele)
		if ent, ok2 := ele.Value.(*watermarkEntry); ok2 {
			return ent.data, true
		}
	}
	return nil, false
}

func (l *watermarkLRU) Put(key string, data []byte) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if ele, ok := l.cache[key]; ok {
		// update existing
		l.ll.MoveToFront(ele)
		if ent, ok2 := ele.Value.(*watermarkEntry); ok2 {
			ent.data = data
		}
		return
	}

	// insert new
	ele := l.ll.PushFront(&watermarkEntry{key: key, data: data})
	l.cache[key] = ele

	// evict if over capacity
	if l.ll.Len() > l.capacity {
		last := l.ll.Back()
		if last != nil {
			l.ll.Remove(last)
			if ent, ok := last.Value.(*watermarkEntry); ok {
				delete(l.cache, ent.key)
			}
		}
	}
}
