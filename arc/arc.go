package arc

type ARC struct {
	hasKey   map[string](*Page)
	t1       *LRU
	b1       *LRU
	t2       *LRU
	b2       *LRU
	maxPages int
	p        int
	hits     int
	misses   int
}

func NewARC(limit int) *ARC {
	return &ARC{
		make(map[string](*Page)),
		NewLru(limit),
		NewLru(limit),
		NewLru(limit),
		NewLru(limit),
		limit,
		0,
		0,
		0,
	}
}

func (arc *ARC) MaxStorage() int {
	return arc.maxPages
}

func (arc *ARC) RemainingStorage() int {
	return arc.maxPages - arc.t1.Len() - arc.t2.Len()
}

func (arc *ARC) Len() int {
	return arc.t1.Len() + arc.t2.Len()
}

func (arc *ARC) Get(key string) (*Page, bool) {
	addr, ok := arc.hasKey[key]

	if !ok {
		arc.misses++
		return nil, false
	}

	switch addr.where {
	case t1:
		arc.t1.Remove(key) // remove from t1 and move to t2
		addr.where = t2
		arc.t2.Set(key, addr)
	case t2:
		arc.t2.Get(key) // update position in t2
	default:
		arc.misses++
		return nil, false
	}

	arc.hits++
	return addr, true
}

func (arc *ARC) Set(key string, value *Page) bool {
	addr, ok := arc.hasKey[key]
	if ok {
		switch addr.where {
		case t1: // Case I
			arc.t1.Remove(key)
		case b1: // Case II
			arc.p = min(arc.p+max(1, arc.b2.length/arc.b1.length), arc.maxPages) // adapt
			if arc.isFull() {
				arc.replace(addr)
			}
			arc.b1.Remove(key)
			delete(arc.hasKey, key)
		case b2: // Case III
			arc.p = max(arc.p-max(1, arc.b1.length/arc.b2.length), 0) // adapt
			if arc.isFull() {
				arc.replace(addr)
			}
			arc.b2.Remove(key)
			delete(arc.hasKey, key)
		default:
		}

		value.where = t2
		arc.t2.Set(key, value)
		arc.hasKey[key] = value
		return true
	}

	// case IV from the paper
	if arc.t1.length+arc.b1.length == arc.maxPages {
		if arc.t1.length < arc.maxPages {
			key1, _ := arc.b1.evict()
			delete(arc.hasKey, key1)
			arc.replace(value)
		} else {
			key1, _ := arc.t1.evict()
			delete(arc.hasKey, key1)
		}
	} else {
		if arc.t1.length+arc.b1.length+arc.t2.length+arc.b2.length >= arc.maxPages {
			if arc.t1.length+arc.b1.length+arc.t2.length+arc.b2.length == 2*arc.maxPages {
				key1, _ := arc.b2.evict()
				delete(arc.hasKey, key1)
			}
			arc.replace(value)
		}
	}

	value.where = t1
	arc.t1.Set(key, value)
	arc.hasKey[key] = value
	return true
}

func (arc *ARC) Remove(key string) (value *Page, ok bool) {
	addr, ok := arc.hasKey[key]

	if !ok {
		return nil, false
	}

	switch addr.where {
	case t1:
		arc.t1.Remove(key)
		addr.where = b1
		arc.b1.Set(key, addr)
	case t2:
		arc.t2.Remove(key)
		addr.where = b2
		arc.b2.Set(key, addr)
	default:
		return nil, false
	}

	return addr, true
}

func (arc *ARC) Stats() *Stats {
	return &Stats{arc.hits, arc.misses}
}

func (arc *ARC) replace(addr *Page) {
	if arc.t1.Len() > 0 && (arc.t1.Len() > arc.p || (arc.t1.Len() == arc.p && addr.where == b2)) {
		key, evictPage := arc.t1.evict()
		arc.b1.Set(key, evictPage)
	} else {
		key, evictPage := arc.t2.evict()
		arc.b2.Set(key, evictPage)
	}
}

func (arc *ARC) isFull() bool {
	return arc.RemainingStorage() <= 0
}
