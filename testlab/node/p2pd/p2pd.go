package p2pd

import (
	"fmt"
	"path/filepath"

	"io/ioutil"
	"os"

	capi "github.com/hashicorp/consul/api"
	napi "github.com/hashicorp/nomad/api"
	"github.com/libp2p/go-libp2p-daemon/p2pclient"
	"github.com/libp2p/testlab/utils"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/sirupsen/logrus"
)

type Node struct{}

func (n *Node) Task(options utils.NodeOptions) (*napi.Task, error) {
	task := napi.NewTask("p2pd", "raw_exec")
	command := "/vagrant/bin/p2pd"
	args := []string{
		"-listen", "/ip4/${NOMAD_IP_p2pd}/tcp/${NOMAD_PORT_p2pd}",
		"-hostAddrs", "/ip4/${NOMAD_IP_libp2p}/tcp/${NOMAD_PORT_libp2p}",
		"-metricsAddr", "${NOMAD_ADDR_metrics}",
		"-pubsub",
	}

	if router, ok := options.String("PubsubRouter"); ok {
		args = append(args, "-pubsubRouter", router)
	}

	res := napi.DefaultResources()
	bandwidth := 5
	res.Networks = []*napi.NetworkResource{
		&napi.NetworkResource{
			DynamicPorts: []napi.Port{
				napi.Port{Label: "libp2p"},
				napi.Port{Label: "p2pd"},
				napi.Port{Label: "metrics"},
			},
			MBits: &bandwidth,
		},
	}
	mem := 20
	res.MemoryMB = &mem
	cpu := 20
	res.CPU = &cpu
	task.Require(res)

	metricsSvc := &napi.Service{
		Name:        "metrics",
		PortLabel:   "metrics",
		AddressMode: "host",
	}
	p2pdSvc := &napi.Service{
		Name:        "p2pd",
		PortLabel:   "p2pd",
		AddressMode: "host",
	}
	libp2pSvc := &napi.Service{
		Name:        "libp2p",
		PortLabel:   "libp2p",
		AddressMode: "host",
	}
	task.Services = append(task.Services, metricsSvc, p2pdSvc, libp2pSvc)

	url := ""

	if cid, ok := options.String("Cid"); ok {
		url = fmt.Sprintf("https://gateway.ipfs.io/ipfs/%s", cid)
	}

	if urlOpt, ok := options.String("Fetch"); ok {
		url = urlOpt
	}

	if url != "" {
		task.Artifacts = []*napi.TaskArtifact{
			&napi.TaskArtifact{
				GetterSource: utils.StringPtr(url),
				RelativeDest: utils.StringPtr("p2pd"),
			},
		}
		command = "p2pd"
	}

	if tags, ok := options.StringSlice("Tags"); ok {
		for _, service := range task.Services {
			service.Tags = tags
		}
	}

	if bootstrap, ok := options["Bootstrap"]; ok {
		tmpl := `BOOTSTRAP_PEERS={{range $index, $service := service "%s.libp2p"}}{{if ne $index 0}},{{end}}/ip4/{{$service.Address}}/tcp/{{$service.Port}}/p2p/{{printf "/peerids/ip4/%%s/tcp/%%d" $service.Address $service.Port | key}}{{end}}`
		tmpl = fmt.Sprintf(tmpl, bootstrap)
		env := true
		changeMode := "noop"
		template := &napi.Template{
			ChangeMode:   &changeMode,
			EmbeddedTmpl: &tmpl,
			DestPath:     utils.StringPtr("bootstrap_peers.env"),
			Envvars:      &env,
		}
		task.Templates = append(task.Templates, template)
		args = append(args, "-b", "-bootstrapPeers", "${BOOTSTRAP_PEERS}")
	}

	task.SetConfig("command", command)
	task.SetConfig("args", args)

	return task, nil
}

func (n *Node) PostDeploy(consul *capi.Client, options utils.NodeOptions) error {
	tags, ok := options.StringSlice("Tags")
	if !ok {
		logrus.Info("skipping post deploy for p2pd, no Tags option")
		return nil
	}

	svcs, _, err := consul.Catalog().ServiceMultipleTags("p2pd", tags, nil)
	if err != nil {
		return err
	}
	bootstrapControlAddrs := make([]ma.Multiaddr, len(svcs))
	for i, svc := range svcs {
		addrStr := fmt.Sprintf("/ip4/%s/tcp/%d", svc.ServiceAddress, svc.ServicePort)
		addr, err := ma.NewMultiaddr(addrStr)
		if err != nil {
			return err
		}
		bootstrapControlAddrs[i] = addr
	}
	for _, addr := range bootstrapControlAddrs {
		dir, err := ioutil.TempDir(os.TempDir(), "daemon_client")
		if err != nil {
			return err
		}
		sockPath := filepath.Join("/unix", dir, "ignore.sock")
		listenAddr, _ := ma.NewMultiaddr(sockPath)
		client, err := p2pclient.NewClient(addr, listenAddr)
		if err != nil {
			return err
		}
		defer func() {
			client.Close()
			os.RemoveAll(dir)
		}()
		peerID, addrs, err := client.Identify()
		if err != nil {
			return err
		}
		for _, addr := range addrs {
			kv := &capi.KVPair{
				Key:   fmt.Sprintf("peerids%s", addr.String()),
				Value: []byte(peerID.Pretty()),
			}
			_, err = consul.KV().Put(kv, nil)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
