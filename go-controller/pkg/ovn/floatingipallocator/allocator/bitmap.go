package allocator

import (
	"errors"
	"math/big"
	"sync"
)

var (
	ErrInvalidRange = errors.New("invalid range")
)

type AllocationBitmap struct {
	max                  int
	lock                 sync.Mutex
	count                int
	limit                int
	available, allocated *big.Int
}

var _ Interface = &AllocationBitmap{}

func NewAllocationMap(offsets []int) *AllocationBitmap {
	bitmap := &AllocationBitmap{
		max:       len(offsets),
		lock:      sync.Mutex{},
		count:     0,
		limit:     0,
		available: big.NewInt(0),
		allocated: big.NewInt(0),
	}

	for _, offset := range offsets {
		if offset > bitmap.limit {
			bitmap.limit = offset
		}
		bitmap.available = bitmap.available.SetBit(bitmap.available, offset, 1)
	}
	return bitmap
}

func (r *AllocationBitmap) Allocate(offset int) (bool, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.available.Bit(offset) == 0 {
		return false, nil
	}
	r.available = r.available.SetBit(r.available, offset, 0)
	r.allocated = r.allocated.SetBit(r.allocated, offset, 1)
	r.count++
	return true, nil
}

func (r *AllocationBitmap) AllocatePool(beg, end int) (bool, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if beg > end {
		return false, ErrInvalidRange
	}

	for i := beg; i <= end; i++ {
		if r.available.Bit(i) == 0 {
			return false, nil
		}
	}
	for i := beg; i <= end; i++ {
		r.available = r.available.SetBit(r.available, i, 0)
		r.allocated = r.allocated.SetBit(r.allocated, i, 1)
		r.count++
	}
	return true, nil
}

func (r *AllocationBitmap) AllocateNext() (int, bool, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	for i := 0; i <= r.limit; i++ {
		if r.available.Bit(i) == 1 {
			r.available = r.available.SetBit(r.available, i, 0)
			r.allocated = r.allocated.SetBit(r.allocated, i, 1)
			r.count++
			return i, true, nil
		}
	}
	return 0, false, nil
}

func (r *AllocationBitmap) Release(offset int) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.allocated.Bit(offset) == 0 {
		return nil
	}

	r.available = r.available.SetBit(r.available, offset, 1)
	r.allocated = r.allocated.SetBit(r.allocated, offset, 0)
	r.count--
	return nil
}

func (r *AllocationBitmap) ReleasePool(beg, end int) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if beg > end {
		return ErrInvalidRange
	}

	for i := beg; i <= end; i++ {
		if r.allocated.Bit(i) == 0 {
			continue
		}
		r.available = r.available.SetBit(r.available, i, 1)
		r.allocated = r.allocated.SetBit(r.allocated, i, 0)
		r.count--
	}
	return nil
}

func (r *AllocationBitmap) Has(offset int) bool {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.allocated.Bit(offset) == 1 || r.available.Bit(offset) == 1
}

func (r *AllocationBitmap) HasPool(beg, end int) bool {
	r.lock.Lock()
	defer r.lock.Unlock()

	if beg > end {
		return false
	}

	for i := beg; i <= end; i++ {
		if r.allocated.Bit(i) == 0 && r.available.Bit(i) == 0 {
			return false
		}
	}
	return true
}

func (r *AllocationBitmap) Free() int {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.max - r.count
}

func (r *AllocationBitmap) Used() int {
	r.lock.Lock()
	defer r.lock.Unlock()
	return r.count
}
