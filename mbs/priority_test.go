package mbs

import "testing"

func ct(p float32) *target {
	return &target{
		priority: p,
	}
}

func TestPriority(t *testing.T) {
	p := newQueue()
	p.Insert(ct(2.0))
	p.Insert(ct(9.0))
	p.Insert(ct(3.0))
	p.Insert(ct(8.0))
	p.Insert(ct(5.0))
	p.Insert(ct(6.0))
	p.Insert(ct(4.0))
	p.Insert(ct(1.0))
	p.Insert(ct(7.0))

	for i := 0; i < 9; i++ {
		if p.Pop().priority != float32(9-i) {
			t.Error("bad result")
		}
	}
}
