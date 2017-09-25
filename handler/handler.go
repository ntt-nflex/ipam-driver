package handler

import (
	"errors"
	"fmt"
	"net"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/ipam"

	"github.com/ntt-nflex/ipam-driver/db"
)

// IPAMHandler holds the main IPAM Handler object
type IPAMHandler struct {
	db *db.Client
	ns string
}

// NewHandler creates a new handler object
func NewHandler(db *db.Client, ns string) IPAMHandler {
	h := IPAMHandler{
		db: db,
		ns: ns,
	}

	return h
}

// RequestPool handles requests for a new IP Pool
func (h IPAMHandler) RequestPool(request *ipam.RequestPoolRequest) (*ipam.RequestPoolResponse, error) {
	log.Infof("RequestPool Called : %+v", request)

	var networkName string
	if opt, ok := request.Options["network-name"]; ok {
		networkName = opt
	} else {
		return nil, errors.New("network-name is required")
	}
	if request.Pool == "" {
		return nil, errors.New("Pool is required")
	}

	ipAddr, ipNet, err := net.ParseCIDR(request.Pool)
	if err != nil {
		return nil, fmt.Errorf("Pool is invalid: %s", request.Pool)
	}
	log.Infof("Pool: %s %s", ipAddr, ipNet)

	pool, err := h.CreatePool(networkName, *ipNet, request.Options)
	if err != nil {
		return nil, err
	}

	response := ipam.RequestPoolResponse{
		PoolID: pool.ID,
		Pool:   pool.Network.String(),
		Data:   pool.Options,
	}
	log.Infof("RequestPoolResponse : %+v", response)

	return &response, nil
}

// ReleasePool handles requests to release an IP Pool
func (h IPAMHandler) ReleasePool(request *ipam.ReleasePoolRequest) error {
	log.Infof("ReleasePool Called : %+v", request)

	err := h.DeletePool(request.PoolID)
	if err != nil {
		return fmt.Errorf("Failed to delete pool: %s", err)
	}
	return nil
}

// RequestAddress handles requests for a new IP Address
func (h IPAMHandler) RequestAddress(request *ipam.RequestAddressRequest) (*ipam.RequestAddressResponse, error) {
	log.Infof("RequestAddress Called : %+v", request)

	var addr string
	var err error
	pool, err := h.GetPool(request.PoolID)
	if err != nil {
		return nil, err
	}

	addrType := request.Options["RequestAddressType"]
	if addrType == "com.docker.network.gateway" && pool.Options["AllowGatewayIPAssignment"] == "true" {
		log.Infof("Skipping IP Reservation for Gateway address: %s", request.Address)
		addr, err = h.DontReserveIP(pool, request.Address)
		if err != nil {
			log.Infof("RequestAddress failed: %s", err)
			return nil, fmt.Errorf("Failed to reserve ip %s: %s", request.Address, err)
		}
	} else if request.Address != "" {
		addr, err = h.ReserveIP(pool, request.Address)
		if err != nil {
			log.Infof("RequestAddress failed: %s", err)
			return nil, fmt.Errorf("Failed to reserve ip %s: %s", request.Address, err)
		}
	} else {
		addr, err = h.ReserveFreeIP(pool)
		if err != nil {
			log.Infof("RequestAddress failed: %s", err)
			return nil, fmt.Errorf("Failed to reserve ip: %s", err)
		}
	}

	response := ipam.RequestAddressResponse{
		Address: addr,
		Data:    map[string]string{},
	}

	log.Infof("RequestAddress returning %s", addr)
	return &response, nil
}

// ReleaseAddress handles requests to release an IP Address
func (h IPAMHandler) ReleaseAddress(request *ipam.ReleaseAddressRequest) (err error) {
	log.Infof("ReleaseAddress Called: %v", request)
	err = h.ReleaseIP(request.PoolID, request.Address)
	return err
}

// GetCapabilities handles Capabilities request
func (h IPAMHandler) GetCapabilities() (response *ipam.CapabilitiesResponse, err error) {
	log.Infof("GetCapabilities called")

	return &ipam.CapabilitiesResponse{RequiresMACAddress: true}, nil
}

// GetDefaultAddressSpaces handles DefaultAddressSpaces request
func (h IPAMHandler) GetDefaultAddressSpaces() (response *ipam.AddressSpacesResponse, err error) {
	log.Infof("GetDefaultAddressSpaces called")
	return &ipam.AddressSpacesResponse{
		LocalDefaultAddressSpace:  "Local",
		GlobalDefaultAddressSpace: "Global",
	}, nil
}
