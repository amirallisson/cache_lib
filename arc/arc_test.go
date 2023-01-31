package arc

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"gonum.org/v1/gonum/stat/distuv"
)

func Set(arc *ARC, key string, val *Page) {
	arc.Set(key, val)
	Get(arc, key, val)
}

func Get(arc *ARC, key string, expected_val *Page) {
	val1, ok := arc.Get(key)

	if !ok || expected_val != val1 {
		panic("Error in Get()")
	}
}

func Remove(arc *ARC, key string, expected_ok bool, expected_val *Page) {
	val, ok := arc.Remove(key)
	if ok != expected_ok || val != expected_val {
		panic("Error in Remove()")
	}
}

func TestSet(t *testing.T) {
	maxPages := 2
	pga := PageAllocator{maxPages}
	arc := NewARC(maxPages)

	if arc.MaxStorage() != maxPages {
		panic("Error in MaxStorage()")
	}
	// create pages
	pg1 := pga.Allocate()
	pg2 := pga.Allocate()
	pg3 := pga.Allocate()

	// insert pages into cache
	if arc.RemainingStorage() != maxPages {
		panic("Error in RemaningStorage()")
	}
	Set(arc, "key1", pg1)

	if arc.RemainingStorage() != maxPages-1 {
		panic("Error in RemaningStorage()")
	}
	Set(arc, "Key2", pg2)
	if arc.RemainingStorage() != maxPages-2 {
		panic("Error in RemaningStorage()")
	}
	Set(arc, "Key3", pg3)

	if arc.RemainingStorage() != maxPages-2 {
		panic("Error in RemaningStorage()")
	}
}

func TestOverwrite(t *testing.T) {
	maxPages := 2
	pga := PageAllocator{maxPages}
	arc := NewARC(maxPages)

	if arc.MaxStorage() != maxPages {
		panic("Error in MaxStorage()")
	}
	// create pages

	pg1 := pga.Allocate()
	pg2 := pga.Allocate()

	// insert pages into cache
	if arc.RemainingStorage() != maxPages {
		panic("Error in RemaningStorage()")
	}

	Set(arc, "Key1", pg1)
	if arc.RemainingStorage() != maxPages-1 {
		panic("Error in RemaningStorage()")
	}

	Set(arc, "Key1", pg2)
	if arc.RemainingStorage() != maxPages-1 {
		panic("Error in RemaningStorage()")
	}
}

// check that set does not remove when op_cnt <= cache_size
func TestSetDoesNotRemove(t *testing.T) {
	maxPages := 3
	pga := PageAllocator{maxPages}
	arc := NewARC(maxPages)

	if arc.MaxStorage() != maxPages {
		panic("Error in MaxStorage()")
	}
	// create pages
	pg1 := pga.Allocate()
	pg2 := pga.Allocate()
	pg3 := pga.Allocate()

	// insert pages into cache
	arc.Set("key1", pg1)
	arc.Set("key2", pg2)
	arc.Set("key3", pg3)

	Get(arc, "key1", pg1)
	Get(arc, "key2", pg2)
	Get(arc, "key3", pg3)
}

// check that set does remove when op_cnt > cache_size
func TestSetRemovesIfFull(t *testing.T) {
	maxPages := 3
	pga := PageAllocator{maxPages}
	arc := NewARC(maxPages)

	if arc.MaxStorage() != maxPages {
		panic("Error in MaxStorage()")
	}
	// create pages
	pg1 := pga.Allocate()
	pg2 := pga.Allocate()
	pg3 := pga.Allocate()
	pg4 := pga.Allocate()

	// insert pages into cache
	arc.Set("key1", pg1)
	arc.Set("key2", pg2)
	arc.Set("key3", pg3)
	arc.Set("key4", pg4)

	Get(arc, "key2", pg2)
	Get(arc, "key3", pg3)
	Get(arc, "key4", pg4)

	// key1 is evicted
	_, ok := arc.Get("key1")
	if ok {
		panic("Entries are evicted incorrectly")
	}
}

func TestRemove(t *testing.T) {
	maxPages := 3
	pga := PageAllocator{maxPages}
	arc := NewARC(maxPages)

	if arc.MaxStorage() != maxPages {
		panic("Error in MaxStorage()")
	}
	// create pages
	pg1 := pga.Allocate()
	pg2 := pga.Allocate()
	pg3 := pga.Allocate()
	pg4 := pga.Allocate()

	// insert pages into cache
	arc.Set("key1", pg1) // will not be in the cache by the time of Remove()
	arc.Set("key2", pg2)
	arc.Set("key3", pg3)
	arc.Set("key4", pg4)

	Remove(arc, "key0", false, nil) // key0 not in the cache
	Remove(arc, "key1", false, nil) // key1 not in the cache, evicted into B1 earlier
	Remove(arc, "key2", true, pg2)
	Remove(arc, "key3", true, pg3)
	Remove(arc, "key4", true, pg4)
}

func TestScanResist(t *testing.T) {
	fmt.Println("Test ScanResist...")
	maxPages := 3
	pga := PageAllocator{maxPages}
	arc := NewARC(maxPages)

	if arc.MaxStorage() != maxPages {
		panic("Error in MaxStorage()")
	}
	// create pages
	pg1 := pga.Allocate()
	pg2 := pga.Allocate()
	pg3 := pga.Allocate()
	// insert pages into cache -> all should be in T2
	Set(arc, "key1", pg1)
	Set(arc, "key2", pg2)
	Set(arc, "key3", pg3)
	// start scanning
	for i := 0; i < 10; i++ {
		key := "key" + fmt.Sprint(i+10)
		arc.Set(key, pga.Allocate())
	}

	Get(arc, "key1", pg1)
	Get(arc, "key2", pg2)
	Get(arc, "key3", pg3)
}

// test efficiency

func Pareto(cacheType string) {
	// set up memory
	memSize := 10000
	mem := make([]*Page, memSize)
	pga := PageAllocator{0}
	for i := 0; i < memSize; i++ {
		mem[i] = pga.Allocate()
	}

	// set up cache
	cacheSize := 100
	var cache Cache
	if cacheType == "LRU" {
		cache = NewLru(cacheSize)
	} else if cacheType == "ARC" {
		cache = NewARC(cacheSize)
	} else {
		return
	}

	// set up pareto
	var par distuv.Pareto
	par.Alpha = 0.1
	par.Xm = 1.0

	// set up coin toss
	var un distuv.Uniform
	un.Max = 1
	un.Min = 0
	total := 0

	timer := time.NewTimer(2 * time.Second)
	var stop int32 = 1

	go func() {
		<-timer.C
		atomic.StoreInt32(&stop, 0)
	}()

	for atomic.LoadInt32(&stop) == 1 {
		index := int(par.Rand())
		if index >= memSize || index < 0 {
			continue
		}

		total++
		pg := mem[index]
		if un.Rand() < 0.5 {
			cache.Get(fmt.Sprintf("%v", index))
		} else {
			cache.Set(fmt.Sprintf("%v", index), pg)
		}
	}

	stats := cache.Stats()

	fmt.Printf("-------------------------------------------------\n")
	fmt.Printf("%v tracing...\nTotal Requests = %v\nHits = %v\nMisses = %v\nHitRate = %v\n", cacheType, total, stats.Hits, stats.Misses, 100.0*float32(stats.Hits)/float32(total))
	fmt.Printf("-------------------------------------------------\n")
}

func TestParetoARC(t *testing.T) {
	Pareto("ARC")
}

func TestParetoLRU(t *testing.T) {
	Pareto("LRU")
}
