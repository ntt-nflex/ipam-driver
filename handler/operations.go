package handler

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/coreos/etcd/client"
)

// Pool represents an IP pool
type Pool struct {
	ID      string
	Network net.IPNet
	Options map[string]string
}

// CreatePool creates a new pool
func (h IPAMHandler) CreatePool(name string, network net.IPNet, options map[string]string) (*Pool, error) {
	key := fmt.Sprintf("%s/pool/%s", h.ns, name)
	pool := Pool{
		ID:      name,
		Network: network,
		Options: options,
	}
	data, err := json.Marshal(pool)
	if err != nil {
		return nil, err
	}

	err = h.db.SetKey(key, string(data))
	if err != nil {
		return nil, err
	}
	return &pool, nil
}

// DeletePool creates a new pool
func (h IPAMHandler) DeletePool(name string) error {
	key := fmt.Sprintf("%s/pool/%s", h.ns, name)
	if !h.db.IsKeyExist(key) {
		return nil
	}

	return h.db.DeleteKey(key)
}

// GetPool ...
func (h IPAMHandler) GetPool(poolID string) (*Pool, error) {
	key := fmt.Sprintf("%s/pool/%s", h.ns, poolID)
	data, err := h.db.GetKey(key)
	if err != nil {
		return nil, err
	}
	pool := Pool{}
	err = json.Unmarshal([]byte(data), &pool)
	return &pool, err
}

// ReserveIP reserves an ip address in a pool, if available
func (h IPAMHandler) ReserveIP(pool *Pool, ip string) (string, error) {
	key := fmt.Sprintf("%s/pool/allocated/%s/%s", h.ns, pool.ID, ip)
	err := h.db.SetKeyIfNotExist(key, "1")
	if err != nil {
		return "", err
	}
	prefixSize, _ := pool.Network.Mask.Size()
	addr := fmt.Sprintf("%s/%d", ip, prefixSize)
	return addr, nil
}

// ReserveFreeIP ...
func (h IPAMHandler) ReserveFreeIP(pool *Pool) (string, error) {
	// Get Allocated IPs
	key := fmt.Sprintf("%s/pool/allocated/%s", h.ns, pool.ID)
	nodes, err := h.db.GetKeys(key)
	if err != nil {
		// KeyNotFound -> this is the first allocation from the pool - ignore the error
		if !errorIs(client.ErrorCodeKeyNotFound, err) {
			return "", err
		}
	}

	// Make a lookup table
	used := map[string]bool{}
	for _, n := range nodes {
		parts := strings.Split(n.Key, "/")
		ip := parts[len(parts)-1]
		used[ip] = true
	}

	// Get Available IPs - https://gist.github.com/kotakanbe/d3059af990252ba89a82
	nw := pool.Network

	// start with the *second* IP in the network - the first one refers to the network itself
	netip := inc(nw.IP.Mask(nw.Mask))

	// while the *next* IP is in the network - we want to skip the last IP, since that's the broadcast address
	for ; nw.Contains(inc(netip)); netip = inc(netip) {
		ip := netip.String()

		if _, ok := used[ip]; !ok {
			addr, err := h.ReserveIP(pool, ip)
			if err != nil {
				// NodeExist -> this IP is not free, try the next one
				if errorIs(client.ErrorCodeNodeExist, err) {
					continue
				}
				return "", err
			}
			return addr, nil
		}
	}
	return "", fmt.Errorf("No IPs available")
}

// DontReserveIP skipsreserving an ip address in a pool,
// but still returns a valid response
func (h IPAMHandler) DontReserveIP(pool *Pool, ip string) (string, error) {
	prefixSize, _ := pool.Network.Mask.Size()
	addr := fmt.Sprintf("%s/%d", ip, prefixSize)
	return addr, nil
}

// ReleaseIP releases an ip address in a pool
func (h IPAMHandler) ReleaseIP(poolID string, ip string) error {
	key := fmt.Sprintf("%s/pool/allocated/%s/%s", h.ns, poolID, ip)
	if !h.db.IsKeyExist(key) {
		return nil
	}

	return h.db.DeleteKey(key)
}

// inc increments `ip` by one address, returning it in `ret`
// e.g. 10.0.2.100   -> 10.0.2.101
//      10.0.2.255   -> 10.0.3.0
//      10.0.255.255 -> 10.1.0.0
// `ip` is not modified.
func inc(ip net.IP) (ret net.IP) {
	ret = make(net.IP, len(ip))
	copy(ret, ip)

	for j := len(ret) - 1; j >= 0; j-- {
		ret[j]++
		if ret[j] > 0 {
			break
		}
	}
	return
}

// errorIs returns true if `err` is an etcd error with code `code`.
// returns false if `err` is not an etcd error, or its code is not `code`
func errorIs(code int, err error) bool {
	if err, ok := err.(client.Error); ok {
		return err.Code == code
	}
	return false
}
