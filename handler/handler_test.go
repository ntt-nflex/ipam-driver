package handler_test

import (
	"fmt"
	"os"
	"sort"
	"testing"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/ipam"
	. "github.com/smartystreets/goconvey/convey"

	"github.com/ntt-nflex/ipam-driver/db"
	"github.com/ntt-nflex/ipam-driver/handler"
)

var (
	cli *db.Client
)

const namespace = "/nflex/ipam-test"

func TestMain(m *testing.M) {
	log.SetLevel(log.FatalLevel)
	setup()
	cleanup()
	code := m.Run()
	//shutdown()
	os.Exit(code)
}

func setup() {
	hosts := []string{"http://127.0.0.1:4001"}
	cli = db.NewClient(hosts)
}

func cleanup() {
	log.Info("cleanup")
	// fetch directory
	if cli.IsKeyExist(namespace) {
		log.Info("delete keys")
		nodes, err := cli.GetKeys(namespace)
		if err != nil {
			log.Fatalf("Error reading keys: %s", err)
		}
		// print directory keys
		sort.Sort(nodes)
		for _, n := range nodes {
			log.Infof("Delete Key: %q, Value: %q\n", n.Key, n.Value)
			cli.DeleteKey(n.Key)
		}
	}
	log.Info("cleanup ok")
}

func TestRequestPool(t *testing.T) {
	Convey("When RequestPool is called", t, func() {
		h := handler.NewHandler(cli, namespace)

		name := "test-request-pool"
		addr := "10.0.1.0/24"

		Convey("When a valid Pool and Name are provided, a pool is created", func() {
			request := ipam.RequestPoolRequest{
				Pool:    addr,
				Options: map[string]string{"network-name": name},
			}
			response, err := h.RequestPool(&request)
			So(err, ShouldBeNil)
			So(response.PoolID, ShouldEqual, name)
			So(response.Pool, ShouldEqual, addr)

			key := fmt.Sprintf("%s/pool/%s", namespace, name)
			check := cli.IsKeyExist(key)
			So(check, ShouldBeTrue)

			Convey("When RequestPool is called again with same name, the result is the same", func() {
				response, err := h.RequestPool(&request)
				So(err, ShouldBeNil)
				So(response.PoolID, ShouldEqual, name)
				So(response.Pool, ShouldEqual, addr)
			})
		})

		Convey("When Name is missing, an error is raised", func() {
			request := ipam.RequestPoolRequest{
				Pool: addr,
			}
			response, err := h.RequestPool(&request)

			So(response, ShouldBeNil)
			So(err.Error(), ShouldEqual, "network-name is required")
		})

		Convey("When Pool is missing, an error is raised", func() {
			request := ipam.RequestPoolRequest{
				Options: map[string]string{"network-name": name},
			}
			response, err := h.RequestPool(&request)

			So(response, ShouldBeNil)
			So(err.Error(), ShouldEqual, "Pool is required")
		})

		Convey("When Pool is invalid, an error is raised", func() {
			request := ipam.RequestPoolRequest{
				Pool:    "foo",
				Options: map[string]string{"network-name": name},
			}
			response, err := h.RequestPool(&request)

			So(response, ShouldBeNil)
			So(err.Error(), ShouldEqual, "Pool is invalid: foo")
		})
	})
}

func TestReleasePool(t *testing.T) {
	Convey("When ReleasePool is called", t, func() {
		h := handler.NewHandler(cli, namespace)

		name := "test-release-pool"
		addr := "10.0.1.0/24"

		Convey("When a the Pool exists, it will be deleted", func() {
			req1 := ipam.RequestPoolRequest{
				Pool:    addr,
				Options: map[string]string{"network-name": name},
			}
			response, err := h.RequestPool(&req1)
			So(err, ShouldBeNil)

			key := fmt.Sprintf("%s/pool/%s", namespace, response.PoolID)
			check := cli.IsKeyExist(key)
			So(check, ShouldBeTrue)

			request := ipam.ReleasePoolRequest{
				PoolID: response.PoolID,
			}
			err = h.ReleasePool(&request)
			So(err, ShouldBeNil)

			check = cli.IsKeyExist(key)
			So(check, ShouldBeFalse)
		})

		Convey("When a the Pool does not exist, the result is the same", func() {
			request := ipam.ReleasePoolRequest{
				PoolID: name,
			}
			err := h.ReleasePool(&request)
			So(err, ShouldBeNil)

			key := fmt.Sprintf("%s/pool/%s", namespace, name)
			check := cli.IsKeyExist(key)
			So(check, ShouldBeFalse)
		})
	})
}

func TestRequestAddress(t *testing.T) {
	Convey("When RequestAddress is called", t, func() {
		Reset(cleanup)

		h := handler.NewHandler(cli, namespace)

		name := "test-release-pool"
		addr := "10.0.1.0/24"
		ip := "10.0.1.100"

		req1 := ipam.RequestPoolRequest{
			Pool:    addr,
			Options: map[string]string{"network-name": name},
		}
		_, err := h.RequestPool(&req1)
		So(err, ShouldBeNil)

		Convey("When a fixed ip is requested, the ip will be allocated", func() {
			request := ipam.RequestAddressRequest{
				PoolID:  name,
				Address: ip,
			}
			response, err := h.RequestAddress(&request)
			So(err, ShouldBeNil)
			So(response.Address, ShouldEqual, "10.0.1.100/24")

			key := fmt.Sprintf("%s/pool/allocated/%s/%s", namespace, name, ip)
			check := cli.IsKeyExist(key)
			So(check, ShouldBeTrue)

			Convey("When the same fixed IP is requested again, we get an error", func() {
				request := ipam.RequestAddressRequest{
					PoolID:  name,
					Address: ip,
				}
				response, err := h.RequestAddress(&request)
				So(err, ShouldNotBeNil)
				So(response, ShouldBeNil)
			})
		})

		Convey("When address is not specified, an ip will be allocated dynamically", func() {
			request := ipam.RequestAddressRequest{
				PoolID: name,
			}
			response1, err := h.RequestAddress(&request)
			So(err, ShouldBeNil)
			So(response1.Address, ShouldNotBeEmpty)
			So(response1.Address, ShouldStartWith, "10.0.1.")
			So(response1.Address, ShouldEndWith, "/24")

			key := fmt.Sprintf("%s/pool/allocated/%s/%s", namespace, name, response1.Address)
			check := cli.IsKeyExist(key)
			So(check, ShouldBeTrue)

			Convey("When the request is repeated, another ip should be allocated", func() {
				response2, err := h.RequestAddress(&request)
				So(err, ShouldBeNil)
				So(response2.Address, ShouldNotBeEmpty)
				So(response2.Address, ShouldStartWith, "10.0.1.")
				So(response2.Address, ShouldEndWith, "/24")
				So(response2.Address, ShouldNotEqual, response1.Address)
			})
		})

		Convey("When the address space is exhausted, IP allocation should fail", func() {

			name := "test-exhaust-pool"
			addr := "10.0.1.0/28"

			req1 := ipam.RequestPoolRequest{
				Pool:    addr,
				Options: map[string]string{"network-name": name},
			}
			_, err := h.RequestPool(&req1)
			So(err, ShouldBeNil)

			request := ipam.RequestAddressRequest{
				PoolID: name,
			}

			for i := 0; i < 14; i++ {
				response, err := h.RequestAddress(&request)
				So(err, ShouldBeNil)
				So(response.Address, ShouldNotBeEmpty)
				So(response.Address, ShouldStartWith, "10.0.1.")
				So(response.Address, ShouldEndWith, "/28")
			}

			_, err = h.RequestAddress(&request)
			So(err, ShouldNotBeNil)

			Convey("When an IP is returned to the pool, the next allocation request should receive that IP", func() {
				err = h.ReleaseAddress(&ipam.ReleaseAddressRequest{
					PoolID:  name,
					Address: "10.0.1.3",
				})
				So(err, ShouldBeNil)

				response, err := h.RequestAddress(&request)
				So(err, ShouldBeNil)
				So(response.Address, ShouldEqual, "10.0.1.3/28")

				Convey("If another request is made, the allocation should fail again", func() {
					_, err = h.RequestAddress(&request)
					So(err, ShouldNotBeNil)
				})
			})
		})
	})
}

func TestRequestGatewayAddress(t *testing.T) {
	Convey("When RequestAddress is called", t, func() {
		Reset(cleanup)

		h := handler.NewHandler(cli, namespace)

		name := "test-release-pool"
		addr := "10.0.1.0/24"
		ip := "10.0.1.1"

		req1 := ipam.RequestPoolRequest{
			Pool: addr,
			Options: map[string]string{"network-name": name,
				"AllowGatewayIPAssignment": "true"},
		}
		_, err := h.RequestPool(&req1)
		So(err, ShouldBeNil)

		Convey("When a gateway ip is requested, response is as expected", func() {
			request := ipam.RequestAddressRequest{
				PoolID:  name,
				Address: ip,
				Options: map[string]string{"RequestAddressType": "com.docker.network.gateway"},
			}
			response, err := h.RequestAddress(&request)
			So(err, ShouldBeNil)
			So(response.Address, ShouldEqual, "10.0.1.1/24")

			//check := cli.IsKeyExist(key)
			//So(check, ShouldBeFalse)

			Convey("When the fixed gateway ip is requested, the ip will be allocated", func() {
				request := ipam.RequestAddressRequest{
					PoolID:  name,
					Address: ip,
				}
				response, err := h.RequestAddress(&request)
				So(err, ShouldBeNil)
				So(response.Address, ShouldEqual, "10.0.1.1/24")

				//check = cli.IsKeyExist(key)
				//So(check, ShouldBeTrue)

				Convey("When the same fixed IP is requested again, we get an error", func() {
					request := ipam.RequestAddressRequest{
						PoolID:  name,
						Address: ip,
					}
					response, err := h.RequestAddress(&request)
					So(err, ShouldNotBeNil)
					So(response, ShouldBeNil)
				})
			})
		})
	})
}
