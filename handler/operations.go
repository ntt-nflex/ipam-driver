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
	Data    map[string]string
}

// CreatePool creates a new pool
func (h IPAMHandler) CreatePool(name string, network net.IPNet) (*Pool, error) {
	key := fmt.Sprintf("%s/pool/%s", h.ns, name)
	pool := Pool{
		ID:      name,
		Network: network,
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
func (h IPAMHandler) ReserveIP(poolID string, ip string) (string, error) {
	pool, err := h.GetPool(poolID)
	if err != nil {
		return "", err
	}

	key := fmt.Sprintf("%s/pool/allocated/%s/%s", h.ns, poolID, ip)
	err = h.db.SetKeyIfNotExist(key, "1")
	if err != nil {
		return "", err
	}
	prefixSize, _ := pool.Network.Mask.Size()
	addr := fmt.Sprintf("%s/%d", ip, prefixSize)
	return addr, nil
}

// ReserveFreeIP ...
func (h IPAMHandler) ReserveFreeIP(poolID string) (string, error) {
	//Get Pool
	key := fmt.Sprintf("%s/pool/%s", h.ns, poolID)
	data, err := h.db.GetKey(key)
	if err != nil {
		return "", err
	}
	pool := Pool{}
	err = json.Unmarshal([]byte(data), &pool)
	if err != nil {
		return "", err
	}

	// Get Allocated IPs
	key = fmt.Sprintf("%s/pool/%s/allocated", h.ns, poolID)
	nodes, err := h.db.GetKeys(key)
	if err != nil {
		return "", err
	}

	// Make a lookup table
	used := map[string]bool{}
	for _, n := range nodes {
		parts := strings.Split(n.Key, "/")
		ip := parts[len(parts)-1]
		used[ip] = true
	}

	// Get Available IPs - https://gist.github.com/kotakanbe/d3059af990252ba89a82
	var ips []string
	nw := pool.Network
	for ip := nw.IP.Mask(nw.Mask); nw.Contains(ip); inc(ip) {
		ips = append(ips, ip.String())
	}
	// remove network address and broadcast address
	ips = ips[1 : len(ips)-1]
	for _, ip := range ips {
		if _, ok := used[ip]; !ok {
			addr, err := h.ReserveIP(poolID, ip)
			if err != nil {
				if err, ok := err.(client.Error); ok {
					if err.Code == client.ErrorCodeNodeExist {
						continue
					}
				}
				return "", err
			}
			return addr, nil
		}
	}
	return "", fmt.Errorf("No IPs available")
}

// ReleaseIP releases an ip address in a pool
func (h IPAMHandler) ReleaseIP(poolID string, ip string) error {
	key := fmt.Sprintf("%s/pool/allocated/%s/%s", h.ns, poolID, ip)
	if !h.db.IsKeyExist(key) {
		return nil
	}

	return h.db.DeleteKey(key)
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
