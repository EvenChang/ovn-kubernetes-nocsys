package floatingipallocator

import (
	"errors"
	"fmt"
	"math/big"
	"net"
	"strings"

	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/ovn/floatingipallocator/allocator"
	utilnet "k8s.io/utils/net"
)

var (
	ErrFull              = errors.New("range is full")
	ErrAllocated         = errors.New("provided IP is already allocated")
	ErrInvalidRange      = errors.New("invalid ip range")
)

type Interface interface {
	Allocate(net.IP) error
	AllocatePool(net.IP, net.IP) error
	AllocateNext() (net.IP, error)
	Release(net.IP) error
	ReleasePool(net.IP, net.IP) error
	ForEach(func(net.IP))

	Has(net.IP) bool
	HasPool(net.IP, net.IP) bool
}

type ErrNotInRange struct {
	ip string
}

func (e *ErrNotInRange) Error() string {
	return fmt.Sprintf("provided ip %s is not in the valid range.", e.ip)
}

type Range struct {
	base *big.Int
	alloc allocator.Interface
}

type ipAddrRange struct {
	start *big.Int
	stop *big.Int
}

func NewAllocatorRange(addrs []string) (*Range, error) {
	base := big.NewInt(0)
	var ranges []*ipAddrRange
	for _, addr := range addrs {
		ips := strings.Split(addr, ",")
		if len(ips) == 1 {
			start := utilnet.BigForIP(net.ParseIP(ips[0]))
			ranges = append(ranges, &ipAddrRange{
				start: start,
				stop: start,
			})

			if base.Int64() == 0 || base.Cmp(start) == 1 {
				base = start
			}
		} else if len(ips) == 2 {
			start := utilnet.BigForIP(net.ParseIP(ips[0]))
			stop := utilnet.BigForIP(net.ParseIP(ips[1]))
			if stop.Cmp(start) < 0 {
				return nil, ErrInvalidRange
			}
			ranges = append(ranges, &ipAddrRange{
				start: start,
				stop: stop,
			})
			if base.Int64() == 0 || base.Cmp(start) == 1 {
				base = start
			}
		}
	}

	var offsets []int
	for _, addr := range ranges {
		start := int(big.NewInt(0).Sub(addr.start, base).Int64())
		stop := int(big.NewInt(0).Sub(addr.stop, base).Int64())
		for i := start; i <= stop; i++ {
			offsets = append(offsets, i)
		}
	}

	return &Range{
		base:  base,
		alloc: allocator.NewAllocationMap(offsets),
	}, nil
}

func (r *Range) Free() int {
	return r.alloc.Free()
}

func (r *Range) Used() int {
	return r.alloc.Used()
}

func (r *Range) Allocate(ip net.IP) error {
	ok, offset := r.contains(ip)
	if !ok {
		return &ErrNotInRange{ip.String()}
	}

	allocated, err := r.alloc.Allocate(offset)
	if err != nil {
		return err
	}
	if !allocated {
		return ErrAllocated
	}
	return nil
}

func (r *Range) AllocatePool(begIp net.IP, endIp net.IP) error {
	ok, beg := r.contains(begIp)
	if !ok {
		return &ErrNotInRange{begIp.String()}
	}

	ok, end := r.contains(endIp)
	if !ok {
		return &ErrNotInRange{endIp.String()}
	}

	allocated, err := r.alloc.AllocatePool(beg, end)
	if err != nil {
		return err
	}
	if !allocated {
		return ErrAllocated
	}
	return nil
}

func (r *Range) AllocateNext() (net.IP, error) {
	offset, ok, err := r.alloc.AllocateNext()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrFull
	}
	return utilnet.AddIPOffset(r.base, offset), nil
}

func (r *Range) Release(ip net.IP) error {
	ok, offset := r.contains(ip)
	if !ok {
		return nil
	}

	return r.alloc.Release(offset)
}

func (r *Range) ReleasePool(begIp net.IP, endIp net.IP) error {
	ok, beg := r.contains(begIp)
	if !ok {
		return &ErrNotInRange{begIp.String()}
	}

	ok, end := r.contains(endIp)
	if !ok {
		return &ErrNotInRange{endIp.String()}
	}

	return r.alloc.ReleasePool(beg, end)
}

func (r *Range) ForEach(fn func(net.IP)) {
	r.alloc.ForEach(func(offset int) {
		fn(utilnet.AddIPOffset(r.base, offset))
	})
}

func (r *Range) Has(ip net.IP) bool {
	ok, _ := r.contains(ip)
	return ok
}

func (r *Range) HasPool(begIp, endIp net.IP) bool {
	ok, beg := r.contains(begIp)
	if !ok {
		return false
	}

	ok, end := r.contains(endIp)
	if !ok {
		return false
	}

	return r.alloc.HasPool(beg, end)
}

func (r *Range) contains(ip net.IP) (bool, int) {
	offset := calculateIPOffset(r.base, ip)
	if r.alloc.Has(offset) {
		return true, offset
	}
	return false, 0
}

func calculateIPOffset(base *big.Int, ip net.IP) int {
	return int(big.NewInt(0).Sub(utilnet.BigForIP(ip), base).Int64())
}