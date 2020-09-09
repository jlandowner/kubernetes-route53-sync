package main

import (
	"context"
	"flag"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
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
	TTL             string
	DNSName         string
	HostedZoneID    string
	UseInternalIP   bool
	NodeSelector    string
	EnableDNSAccess bool
}{
	TTL:             os.Getenv("DNS_TTL"),
	DNSName:         os.Getenv("DNS_NAME"),
	HostedZoneID:    os.Getenv("HOSTEDZONE_ID"),
	UseInternalIP:   os.Getenv("USE_INTERNAL_IP") == "1",
	NodeSelector:    os.Getenv("NODE_SELECTOR"),
	EnableDNSAccess: os.Getenv("ENABLE_DNS_ACCESS") == "1",
}

func main() {
	flag.StringVar(&options.DNSName, "dns-name", options.DNSName, "the dns name for the nodes, comma-separated for multiple (same root)")
	flag.StringVar(&options.TTL, "ttl", options.TTL, "ttl for dns (default 300)")
	flag.StringVar(&options.HostedZoneID, "hostedzone-id", options.HostedZoneID, "Route53 hostedzone id")
	flag.BoolVar(&options.UseInternalIP, "use-internal-ip", options.UseInternalIP, "use internal ips too if external ip's are not available")
	flag.StringVar(&options.NodeSelector, "node-selector", options.NodeSelector, "node selector query")
	flag.BoolVar(&options.EnableDNSAccess, "enable-dns-access", options.EnableDNSAccess, "set false if you cannot resolve name in cluster")
	flag.Parse()

	dnsNames := strings.Split(options.DNSName, ",")
	if len(dnsNames) == 1 && dnsNames[0] == "" {
		flag.Usage()
		log.Fatalln("dns name is required")
	}
	log.Println("DNS Name to sync", dnsNames)

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
	log.Println("DNS record ttl", ttl)

	log.Println("HostedZoneID", options.HostedZoneID)
	log.Println("Enable DNS Access", options.EnableDNSAccess)

	nodeAddressType := core_v1.NodeExternalIP
	if options.UseInternalIP {
		nodeAddressType = core_v1.NodeInternalIP
	}
	log.Println("Node Address Type", nodeAddressType)

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
	sig := make(chan os.Signal, 1)
	defer close(sig)
	signal.Notify(sig, os.Interrupt, syscall.SIGKILL, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())

	nodeSelector := labels.NewSelector()
	if options.NodeSelector != "" {
		selector, err := labels.Parse(options.NodeSelector)
		if err != nil {
			log.Printf("node selector is invalid: %v\n", err)
		} else {
			nodeSelector = selector
		}
	}
	log.Println("NodeSelector", nodeSelector)

	factory := informers.NewSharedInformerFactory(client, time.Minute)
	lister := factory.Core().V1().Nodes().Lister()
	var lastIPs []string
	resync := func() {
		// log.Println("resyncing")
		nodes, err := lister.List(nodeSelector)
		if err != nil {
			log.Println("failed to list nodes", err)
		}

		var ips []string
		for _, node := range nodes {
			if nodeIsReady(node) {
				for _, addr := range node.Status.Addresses {
					if addr.Type == nodeAddressType {
						ips = append(ips, addr.Address)
					}
				}
			}
		}

		if options.EnableDNSAccess {
			currentIPs, err := net.LookupHost(dnsNames[0])
			if err != nil {
				lastIPs = []string{}
			} else {
				sort.Strings(currentIPs)
				lastIPs = currentIPs
			}
		}

		sort.Strings(ips)
		if strings.Join(ips, ",") == strings.Join(lastIPs, ",") {
			log.Println("no change detected")
			return
		}
		log.Println("change detected", "ips:", ips, "lastIPs:", lastIPs)

		// sync Route53
		err = r53.Sync(ctx, ips, dnsNames, int64(ttl), options.HostedZoneID)
		if err != nil {
			log.Println("failed to sync", err)
			return
		}

		lastIPs = ips
		if options.EnableDNSAccess {
			// try to resolve name. if reached to timeout, reset lastIPs
			checkSync(ctx, ips, dnsNames[0], ttl)
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

	go func() {
		informer.Run(stop)
		log.Println("informer stopped")
		cancel()
	}()

	<-sig
	log.Println("termination signal recieved")
	close(stop)

	<-ctx.Done()
	log.Println("well done")
	os.Exit(0)
}

func nodeIsReady(node *core_v1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == core_v1.NodeReady && condition.Status == core_v1.ConditionTrue {
			return true
		}
	}

	return false
}

func checkSync(ctx context.Context, expectIPs []string, dnsName string, ttl int) bool {
	var wait int
	ctx, cancel := context.WithTimeout(ctx, time.Second*time.Duration(ttl*2))
	defer cancel()

	sort.Strings(expectIPs)
	for {
		select {
		case <-ctx.Done():
			log.Println("failed to checkSync. Reached to timeout")
			return false
		default:
			currentIPs, err := net.LookupHost(dnsName)
			log.Printf("lookup host result %v, err %v", currentIPs, err)
			if err == nil {
				sort.Strings(currentIPs)
				if strings.Join(expectIPs, ",") == strings.Join(currentIPs, ",") {
					log.Printf("success to checkSync Name: %v IPs: %v", dnsName, currentIPs)
					return true
				}
			}
			wait += rand.Intn(ttl / 2)
			log.Printf("checking Sync...next check is %d seconds later", wait)
			time.Sleep(time.Second * time.Duration(wait))
		}
	}
}
