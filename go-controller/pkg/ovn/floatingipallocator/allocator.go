package floatingipallocator

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"net"
	"strings"

	"github.com/ovn-org/ovn-kubernetes/go-controller/pkg/ovn/floatingipallocator/allocator"
	utilnet "k8s.io/utils/net"
)

var (
	ErrFull         = errors.New("range is full")
	ErrAllocated    = errors.New("provided IP is already allocated")
	ErrInvalidRange = errors.New("invalid ip range")
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

	Free() int
	Used() int
}

type ErrNotInRange struct {
	ip string
}

func (e *ErrNotInRange) Error() string {
	return fmt.Sprintf("provided ip %s is not in the valid range.", e.ip)
}

type distributor struct {
	base    *big.Int
	alloc   allocator.Interface
	offsets []int
}

type AddrPair struct {
	Begin  net.IP
	End    net.IP
	IsPool bool
}

func NewAddrPair(addr string) (*AddrPair, error) {
	ips := strings.Split(addr, ",")
	if len(ips) == 1 {
		ip := net.ParseIP(ips[0])
		if ip.To4() == nil {
			return nil, fmt.Errorf("%s is not an IPv4 address", ips[0])
		}
		return &AddrPair{ip, ip, false}, nil
	} else if len(ips) == 2 {
		begin := net.ParseIP(ips[0])
		end := net.ParseIP(ips[1])
		if begin.To4() == nil || end.To4() == nil {
			return nil, fmt.Errorf("%s or %s is not an IPv4 address", ips[0], ips[1])
		}
		if bytes.Compare(begin, end) <= 0 {
			return &AddrPair{begin, end, true}, nil
		}
	}
	return nil, fmt.Errorf("bad network address format")
}

func (ap *AddrPair) ToOffset(ip net.IP) []int {
	base := utilnet.BigForIP(ip)
	begin := utilnet.BigForIP(ap.Begin)
	end := utilnet.BigForIP(ap.End)

	// convert to offset
	start := int(big.NewInt(0).Sub(begin, base).Int64())
	stop := int(big.NewInt(0).Sub(end, base).Int64())
	if start < 0 || stop < 0 {
		return nil
	}
	var offsets []int
	for i := start; i <= stop; i++ {
		offsets = append(offsets, i)
	}
	return offsets
}

func NewDistributor(aps []*AddrPair) (*distributor, error) {
	base := aps[0].Begin
	for _, ap := range aps {
		if bytes.Compare(ap.Begin, base) < 0 {
			base = ap.Begin
		}
	}
	var offsets []int
	for _, ap := range aps {
		if offset := ap.ToOffset(base); offset == nil {
			return nil, fmt.Errorf("the offset cannot be calculated")
		} else {
			offsets = append(offsets, offset...)
		}
	}
	return &distributor{
		base:    utilnet.BigForIP(base),
		alloc:   allocator.NewAllocationMap(offsets),
		offsets: offsets,
	}, nil
}

func (r *distributor) Free() int {
	return r.alloc.Free()
}

func (r *distributor) Used() int {
	return r.alloc.Used()
}

func (r *distributor) Allocate(ip net.IP) error {
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

func (r *distributor) AllocatePool(begIp net.IP, endIp net.IP) error {
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

func (r *distributor) AllocateNext() (net.IP, error) {
	offset, ok, err := r.alloc.AllocateNext()
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrFull
	}
	return utilnet.AddIPOffset(r.base, offset), nil
}

func (r *distributor) Release(ip net.IP) error {
	ok, offset := r.contains(ip)
	if !ok {
		return nil
	}

	return r.alloc.Release(offset)
}

func (r *distributor) ReleasePool(begIp net.IP, endIp net.IP) error {
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

func (r *distributor) ForEach(fn func(net.IP)) {
	for _, offset := range r.offsets {
		fn(utilnet.AddIPOffset(r.base, offset))
	}
}

func (r *distributor) Has(ip net.IP) bool {
	ok, _ := r.contains(ip)
	return ok
}

func (r *distributor) HasPool(begIp, endIp net.IP) bool {
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

func (r *distributor) contains(ip net.IP) (bool, int) {
	offset := calculateIPOffset(r.base, ip)
	if offset >= 0 && r.alloc.Has(offset) {
		return true, offset
	}
	return false, 0
}

func calculateIPOffset(base *big.Int, ip net.IP) int {
	return int(big.NewInt(0).Sub(utilnet.BigForIP(ip), base).Int64())
}
