package cluster

import (
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/astranet/astranet"
	"github.com/astranet/astranet/addr"
	"github.com/astranet/astranet/service"
	uuid "github.com/satori/go.uuid"
	log "github.com/sirupsen/logrus"
)

type Cluster struct {
	nodeID   string
	astraNet astranet.AstraNet
}

func NewCluster() *Cluster {
	return &Cluster{
		nodeID:   uuid.Must(uuid.NewV4()).String(),
		astraNet: astranet.New().Router(),
	}
}

func (c *Cluster) NodeID() string {
	return c.nodeID
}

func (c *Cluster) ListenAndServe(address string) (net.Listener, error) {
	l, err := c.astraNet.Bind("", c.nodeID)
	if err != nil {
		return nil, err
	}

	if err = c.astraNet.ListenAndServe("tcp4", address); err != nil {
		return nil, err
	}

	if err = c.astraNet.Join("tcp4", address); err != nil {
		return nil, err
	}

	return l, nil
}

func (c *Cluster) Join(nodes string) error {
	return c.astraNet.Join("tcp4", nodes)
}

func (c *Cluster) Transport() *http.Transport {
	return &http.Transport{
		Dial: c.astraNet.HttpDial,
	}
}

func (c *Cluster) Propagate(uri string) error {
	services := make(map[string]service.ServiceInfo)

	var self service.ServiceInfo

	for _, sv := range c.astraNet.Services() {
		if sv.Upstream == nil || strings.HasPrefix(sv.Service, "ipc.") {
			continue
		}

		if sv.Service == c.nodeID {
			self = sv
			continue
		}

		host, _, _ := net.SplitHostPort(sv.Upstream.RAddr().String())

		if _, ok := services[sv.Service+host]; !ok {
			services[sv.Service] = sv
		}
	}

	for _, sv := range services {
		cli := http.Client{
			Transport: &http.Transport{
				Dial: c.astraNet.HttpDial,
			},
		}

		log.WithFields(log.Fields{
			"node": sv.Service,
			"uri":  uri,
		}).Debug("Propagating to cluster node")

		req, err := http.NewRequest("GET", fmt.Sprintf("http://%s:%d/%s", addr.Uint2Host(sv.Host), sv.Port, uri), nil)
		if err != nil {
			return err
		}

		req.Header.Add("X-Backend", fmt.Sprintf("http://%s:%d", addr.Uint2Host(self.Host), self.Port))

		res, err := cli.Do(req)
		if err != nil {
			return err
		}

		defer res.Body.Close()
	}

	return nil
}
