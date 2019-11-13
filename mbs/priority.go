package mbs

type queue struct {
	data []*target
}

func parent(i int) int { return (i - 1) / 2 }
func left(i int) int   { return 2*i + 1 }
func right(i int) int  { return 2*i + 2 }

func newQueue() *queue {
	return &queue{
		data: make([]*target, 0, 100),
	}
}

func (q *queue) Insert(t *target) {
	q.data = append(q.data, t)
	// make sure to keep heap property:
	for i := len(q.data) - 1; i > 0 && q.data[parent(i)].priority < q.data[i].priority; i = parent(i) {
		q.swap(i, parent(i))
	}
}

func (q *queue) Pop() *target {
	if len(q.data) == 0 {
		return nil
	} else if len(q.data) == 1 {
		t := q.data[0]
		q.data[0] = nil
		q.data = q.data[:0]
		return t
	}

	t := q.data[0]
	q.data[0] = q.data[len(q.data)-1]
	q.data[len(q.data)-1] = nil
	q.data = q.data[:len(q.data)-1]
	q.heapify(0)
	return t
}

func (q *queue) swap(i, j int) {
	q.data[i], q.data[j] = q.data[j], q.data[i]
}

func (q *queue) heapify(i int) {
	l, r := left(i), right(i)
	small := i
	if l < len(q.data) && q.data[l].priority > q.data[i].priority {
		small = l
	}
	if r < len(q.data) && q.data[r].priority > q.data[small].priority {
		small = r
	}
	if small != i {
		q.swap(i, small)
		q.heapify(small)
	}
}
