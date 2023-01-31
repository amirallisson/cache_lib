package arc

const (
	t1 int = 0
	t2 int = 1
	b1 int = 2
	b2 int = 3
	no int = 4
)

type Page struct {
	data  []byte
	where int
}

type PageAllocator struct {
	PageSize int
}

func (pa *PageAllocator) Allocate() *Page {
	return &Page{
		make([]byte, pa.PageSize),
		no,
	}
}
