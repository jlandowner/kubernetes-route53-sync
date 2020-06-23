# kubernetes-route53-sync

[![GoReportCard](https://goreportcard.com/badge/github.com/jlandowner/kubernetes-route53-sync)](https://goreportcard.com/report/github.com/jlandowner/kubernetes-route53-sync)

[![dockeri.co](https://dockeri.co/image/jlandowner/kubernetes-route53-sync)](https://hub.docker.com/r/jlandowner/kubernetes-route53-sync)

Kubernetes controller to synchronize node IPs with Route53 DNS records

This project is forked from https://github.com/calebdoxsey/kubernetes-cloudflare-sync

# Install

Fetch files

```shell
git clone https://github.com/jlandowner/kubernetes-route53-sync.git
```

Modify AWS Congiguration to yours in secret.yaml

```shell
sed -i -e 's/<YOUR_AWS_ACCESS_KEY_ID>/XXXXXX/' ./kubernetes-route53-sync/master/kubernetes/secret.yaml

sed -i -e 's/<YOUR_AWS_SECRET_ACCESS_KEY>/YYYYYY/' ./kubernetes-route53-sync/master/kubernetes/secret.yaml
```

Apply the configuration files

```shell
kubectl apply -f ./kubernetes-route53-sync/kubernetes/
```

# LICENSE
MIT License
