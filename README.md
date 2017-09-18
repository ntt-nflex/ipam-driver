# nflex IPAM Driver for Docker

This project provides a Docker Plugin which implements the IPAM Driver specification.

https://github.com/docker/libnetwork/blob/master/docs/ipam.md


It uses the official docker helper library:

https://github.com/docker/go-plugins-helpers/blob/master/ipam/api.go


This plugin enables the creation of multiple overlay networks with seperate but overlapping IP Pools.

IP Allocation is handled via Etcd, which should be available on the local host.  Default address if http://localhost:4001 and can be configured by the ETCD_HOSTS opiton.


## Usage:

docker network create --driver overlay --ipam-driver nflex/ipam-driver:0.0.1 --ipam-opt="network-name=overlay1" --subnet 10.0.1.0/24 overlay1


## Limitations

- Supports Overlay networks only
- No Subnet support (yet, could be added if need be)


## Docker Hub

https://hub.docker.com/r/nflex/ipam-driver/


## References:

Ideas and code examples borrowed from:

- Infoblox driver: https://github.com/infobloxopen/docker-infoblox
- Rootsonic repo: https://github.com/rootsongjc/docker-ipam-plugin/

Original issue that this solves:

- https://github.com/moby/moby/issues/28375
