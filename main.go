package main

import (
	"context"
	"flag"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	r53 "github.com/jlandowner/kubernetes-route53-sync/pkg/route53"

	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

var options = struct {
	TTL           string
	DNSName       string
	UseInternalIP bool
	NodeSelector  string
}{
	TTL:           os.Getenv("R53_TTL"),
	DNSName:       os.Getenv("DNS_NAME"),
	UseInternalIP: os.Getenv("USE_INTERNAL_IP") != "",
	NodeSelector:  os.Getenv("NODE_SELECTOR"),
}

func main() {
	flag.StringVar(&options.DNSName, "dns-name", options.DNSName, "the dns name for the nodes, comma-separated for multiple (same root)")
	flag.StringVar(&options.TTL, "ttl", options.TTL, "ttl for dns (default 300)")
	flag.BoolVar(&options.UseInternalIP, "use-internal-ip", options.UseInternalIP, "use internal ips too if external ip's are not available")
	flag.StringVar(&options.NodeSelector, "node-selector", options.NodeSelector, "node selector query")
	flag.Parse()

	dnsNames := strings.Split(options.DNSName, ",")
	if len(dnsNames) == 1 && dnsNames[0] == "" {
		flag.Usage()
		log.Fatalln("dns name is required")
	}

	for i, dnsName := range dnsNames {
		if !strings.HasSuffix(dnsName, ".") {
			dnsNames[i] = dnsName + "."
		}
	}

	ttl, err := strconv.Atoi(options.TTL)
	if err != nil {
		log.Println("TTL config not found or incorrect, defaulting to 300")
		ttl = 300
	}

	log.SetOutput(os.Stdout)

	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalln(err)
	}

	client, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalln(err)
	}

	stop := make(chan struct{})
	defer close(stop)

	nodeSelector := labels.NewSelector()
	if options.NodeSelector != "" {
		selector, err := labels.Parse(options.NodeSelector)
		if err != nil {
			log.Printf("node selector is invalid: %v\n", err)
		} else {
			nodeSelector = selector
		}
	}

	factory := informers.NewSharedInformerFactory(client, time.Minute)
	lister := factory.Core().V1().Nodes().Lister()
	var lastIPs []string
	resync := func() {
		log.Println("resyncing")
		nodes, err := lister.List(nodeSelector)
		if err != nil {
			log.Println("failed to list nodes", err)
		}

		var ips []string
		for _, node := range nodes {
			if nodeIsReady(node) {
				for _, addr := range node.Status.Addresses {
					if addr.Type == core_v1.NodeExternalIP {
						ips = append(ips, addr.Address)
					}
				}
			}
		}
		if options.UseInternalIP && len(ips) == 0 {
			for _, node := range nodes {
				if nodeIsReady(node) {
					for _, addr := range node.Status.Addresses {
						if addr.Type == core_v1.NodeInternalIP {
							ips = append(ips, addr.Address)
						}
					}
				}
			}
		}

		sort.Strings(ips)
		log.Println("ips:", ips)
		if strings.Join(ips, ",") == strings.Join(lastIPs, ",") {
			log.Println("no change detected")
			return
		}
		lastIPs = ips

		ctx := context.Background()
		err = r53.Sync(ctx, ips, dnsNames, int64(ttl))
		if err != nil {
			log.Println("failed to sync", err)
		}
	}

	informer := factory.Core().V1().Nodes().Informer()
	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			resync()
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			resync()
		},
		DeleteFunc: func(obj interface{}) {
			resync()
		},
	})
	informer.Run(stop)
	log.Println("shutdown")
	select {}
}

func nodeIsReady(node *core_v1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == core_v1.NodeReady && condition.Status == core_v1.ConditionTrue {
			return true
		}
	}

	return false
}
