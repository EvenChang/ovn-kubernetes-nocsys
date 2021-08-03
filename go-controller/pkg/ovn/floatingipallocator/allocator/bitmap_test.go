package allocator

import "testing"

func TestAllocate(t *testing.T) {
	offsets := []int{3, 4, 5, 6, 8, 20}
	m := NewAllocationMap(offsets)

	if ok, _ := m.AllocatePool(3, 8); ok {
		t.Fatalf("unexpected error")
	}
	if ok, _ := m.AllocatePool(3, 6); !ok {
		t.Fatalf("unexpected error")
	}
	if _, ok, _ := m.AllocateNext(); !ok {
		t.Fatalf("unexpected error")
	}

	if m.count != 5 {
		t.Errorf("expect to get %d, but got %d", 5, m.count)
	}
	if f := m.Free(); f != len(offsets)-5 {
		t.Errorf("expect to get %d, but got %d", len(offsets)-5, f)
	}
}

func TestAllocateMax(t *testing.T) {
	offsets := []int{3, 4, 8, 20}
	m := NewAllocationMap(offsets)
	for i := 0; i < len(offsets); i++ {
		if _, ok, _ := m.AllocateNext(); !ok {
			t.Fatalf("unexpected error")
		}
	}
	if _, ok, _ := m.AllocateNext(); ok {
		t.Errorf("unexpected success")
	}
	if f := m.Free(); f != 0 {
		t.Errorf("expect to get %d, but got %d", 0, f)
	}
}

func TestAllocateError(t *testing.T) {
	offsets := []int{3, 4, 8, 20}
	m := NewAllocationMap(offsets)
	if ok, _ := m.AllocatePool(3, 4); !ok {
		t.Errorf("error allocate offset range 3 ~ 4")
	}
	if ok, _ := m.Allocate(4); ok {
		t.Errorf("unexpected success")
	}
}

func TestHas(t *testing.T) {
	offsets := []int{3, 4, 5, 6, 8, 20}
	m := NewAllocationMap(offsets)
	if ok := m.HasPool(3, 9); ok {
		t.Errorf("unexpected success")
	}
	if ok := m.HasPool(3, 5); !ok {
		t.Errorf("unexpected error")
	}
	if ok := m.Has(8); !ok {
		t.Errorf("unexpected error")
	}
	if ok := m.Has(9); ok {
		t.Errorf("unexpected success")
	}
}

func TestRelease(t *testing.T) {
	offsets := []int{3, 4, 5, 6, 8, 20}
	m := NewAllocationMap(offsets)
	if ok, _ := m.AllocatePool(3, 5); !ok {
		t.Errorf("error allocate offset range [%v, %v]", 3, 5)
	}
	if ok, _ := m.Allocate(8); !ok {
		t.Errorf("unexpected error")
	}
	if ok, _ := m.Allocate(5); ok {
		t.Errorf("expect offset %v allocated", 5)
	}
	if err := m.ReleasePool(3, 8); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if ok, _ := m.Allocate(5); !ok {
		t.Errorf("expect offset %v not allocated", 5)
	}
	if f := m.Free(); f != len(offsets)-1 {
		t.Errorf("expect to get %d, but got %d", len(offsets)-1, f)
	}
}
