package snowflake

import (
	"crypto/rand"
	"encoding/binary"
	"strconv"
	"sync"
	"time"
)

const (
	epoch int64 = 1288834974657

	sequenceBits   = 10
	randomNodeBits = 12

	maxSequence   int64 = -1 ^ (-1 << sequenceBits)
	maxRandomNode int64 = -1 ^ (-1 << randomNodeBits)

	sequenceShift  = randomNodeBits
	timestampShift = sequenceBits + randomNodeBits
)

type idGenerator struct {
	mu         sync.Mutex
	randomNode int64
	lastTime   int64
	sequence   int64
}

var generator *idGenerator

func newRandomNode() int64 {
	var buf [8]byte
	if _, err := rand.Read(buf[:]); err != nil {
		panic(err)
	}

	return int64(binary.BigEndian.Uint64(buf[:])) & maxRandomNode
}

func currentTimeMillis() int64 {
	return time.Now().UnixMilli() - epoch
}

func waitNextMillis(lastTime int64) int64 {
	now := currentTimeMillis()
	for now <= lastTime {
		time.Sleep(time.Millisecond)
		now = currentTimeMillis()
	}
	return now
}

func init() {
	generator = &idGenerator{
		randomNode: newRandomNode(),
		lastTime:   -1,
	}
}

func (g *idGenerator) generate() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()

	now := currentTimeMillis()
	if now < g.lastTime {
		now = waitNextMillis(g.lastTime)
	}

	if now == g.lastTime {
		g.sequence = (g.sequence + 1) & maxSequence
		if g.sequence == 0 {
			now = waitNextMillis(g.lastTime)
		}
	} else {
		g.sequence = 0
	}

	g.lastTime = now

	return (now << timestampShift) |
		(g.sequence << sequenceShift) |
		g.randomNode
}

func Generate() int64 {
	return generator.generate()
}

func GenerateString() string {
	return strconv.FormatInt(Generate(), 10)
}
