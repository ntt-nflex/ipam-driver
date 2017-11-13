package db

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/coreos/etcd/client"
	"golang.org/x/net/context"
)

// Client is a wrapper for the etcd client, and provides our helper methods
type Client struct {
	hosts  []string
	client client.Client
}

// NewClient creates a new Client object
func NewClient(hosts []string) *Client {
	cfg := client.Config{
		Endpoints: hosts,
		Transport: client.DefaultTransport,
		// set timeout per request to fail fast when the target endpoint is unavailable
		HeaderTimeoutPerRequest: time.Second,
	}
	c, err := client.New(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to etcd hosts: %s %s", hosts, err)
	}
	return &Client{
		hosts:  hosts,
		client: c,
	}
}

// GetKey ...
func (c *Client) GetKey(key string) (string, error) {
	kapi := client.NewKeysAPI(c.client)
	resp, err := kapi.Get(context.Background(), key, nil)
	if err != nil {
		return "", err
	}
	log.Debugf("Get key %s with value %s", resp.Node.Key, resp.Node.Value)
	return resp.Node.Value, err
}

//GetKeys ...
func (c *Client) GetKeys(dir string) (client.Nodes, error) {
	kapi := client.NewKeysAPI(c.client)
	resp, err := kapi.Get(context.Background(), dir, &client.GetOptions{Sort: true})
	if err != nil {
		if err, ok := err.(client.Error); ok {
			if err.Code == client.ErrorCodeNotDir {
				return []*client.Node{}, nil
			}
		}

		return nil, err
	}
	log.Infof("Get %d keys from dir %s", len(resp.Node.Nodes), resp.Node.Key)
	return resp.Node.Nodes, nil
}

// IsKeyExist ...
func (c *Client) IsKeyExist(key string) bool {
	kapi := client.NewKeysAPI(c.client)
	_, err := kapi.Get(context.Background(), key, nil)
	if client.IsKeyNotFound(err) == true {
		return false
	}
	return true
}

// SetKey ...
func (c *Client) SetKey(key, value string) error {
	kapi := client.NewKeysAPI(c.client)
	resp, err := kapi.Set(context.Background(), key, value, nil)
	if err != nil {
		return err
	}
	log.Debugf("Set key %s with value %s", resp.Node.Key, resp.Node.Value)
	return nil
}

// SetKeyIfNotExist ...
func (c *Client) SetKeyIfNotExist(key, value string) error {
	kapi := client.NewKeysAPI(c.client)
	opts := &client.SetOptions{
		PrevExist: client.PrevNoExist,
	}
	resp, err := kapi.Set(context.Background(), key, value, opts)
	if err != nil {
		return err
	}
	log.Debugf("Set key %s with value %s", resp.Node.Key, resp.Node.Value)
	return nil
}

// DeleteKey ...
func (c *Client) DeleteKey(key string) error {
	kapi := client.NewKeysAPI(c.client)
	resp, err := kapi.Delete(context.Background(), key, &client.DeleteOptions{Recursive: true})
	if err != nil {
		return err
	}
	log.Debugf("Delete key %s with value %s", resp.Node.Key, resp.Node.Value)
	return err
}
