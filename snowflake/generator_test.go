package snowflake

import (
	"strconv"
	"sync"
	"testing"
)

func decodeTimestamp(id int64) int64 {
	return id >> timestampShift
}

func decodeSequence(id int64) int64 {
	return (id >> sequenceShift) & maxSequence
}

func decodeRandomNode(id int64) int64 {
	return id & maxRandomNode
}

func TestGenerate_BitLayout(t *testing.T) {
	const randomNode int64 = 1234
	g := &idGenerator{
		randomNode: randomNode,
		lastTime:   -1,
	}

	id := g.generate()

	if got := decodeRandomNode(id); got != randomNode {
		t.Fatalf("random node = %d, want %d", got, randomNode)
	}

	if got := decodeSequence(id); got != 0 {
		t.Fatalf("sequence = %d, want 0", got)
	}

	if got := decodeTimestamp(id); got <= 0 {
		t.Fatalf("timestamp = %d, want positive", got)
	}
}

func TestGenerate_IncreasesWithinProcess(t *testing.T) {
	g := &idGenerator{
		randomNode: 1,
		lastTime:   -1,
	}

	last := g.generate()
	for i := 0; i < 10000; i++ {
		id := g.generate()
		if id <= last {
			t.Fatalf("id did not increase at index %d: got %d after %d", i, id, last)
		}
		last = id
	}
}

func TestGenerate_WaitsForNextMillisWhenSequenceOverflows(t *testing.T) {
	lastTime := currentTimeMillis()
	g := &idGenerator{
		randomNode: 2,
		lastTime:   lastTime,
		sequence:   maxSequence,
	}

	id := g.generate()

	if got := decodeSequence(id); got != 0 {
		t.Fatalf("sequence = %d, want 0 after overflow", got)
	}

	if got := decodeTimestamp(id); got <= lastTime {
		t.Fatalf("timestamp = %d, want greater than %d after overflow", got, lastTime)
	}
}

func TestGenerate_ConcurrentUnique(t *testing.T) {
	g := &idGenerator{
		randomNode: 3,
		lastTime:   -1,
	}

	const (
		workers      = 32
		idsPerWorker = 500
		total        = workers * idsPerWorker
	)

	ids := make(chan int64, total)
	var wg sync.WaitGroup

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < idsPerWorker; j++ {
				ids <- g.generate()
			}
		}()
	}

	wg.Wait()
	close(ids)

	seen := make(map[int64]struct{}, total)
	for id := range ids {
		if _, ok := seen[id]; ok {
			t.Fatalf("duplicate id generated: %d", id)
		}
		seen[id] = struct{}{}
	}

	if len(seen) != total {
		t.Fatalf("generated %d unique ids, want %d", len(seen), total)
	}
}

func TestGenerateString(t *testing.T) {
	id := Generate()
	got, err := strconv.ParseInt(GenerateString(), 10, 64)
	if err != nil {
		t.Fatalf("GenerateString returned invalid int64: %v", err)
	}

	if got <= id {
		t.Fatalf("GenerateString id = %d, want greater than previous Generate id %d", got, id)
	}
}
