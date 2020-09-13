# kubernetes-route53-sync

![GoReportCard](https://goreportcard.com/badge/github.com/jlandowner/kubernetes-route53-sync)
[![GoReportCard](https://goreportcard.com/badge/github.com/jlandowner/kubernetes-route53-sync)](https://goreportcard.com/report/github.com/jlandowner/kubernetes-route53-sync)
![DockerPulls](https://img.shields.io/docker/pulls/jlandowner/kubernetes-route53-sync)
![GithubActionsStatus](https://github.com/jlandowner/kubernetes-route53-sync/workflows/release/badge.svg)

[![dockeri.co](https://dockeri.co/image/jlandowner/kubernetes-route53-sync)](https://hub.docker.com/r/jlandowner/kubernetes-route53-sync)

Kubernetes controller to synchronize node IPs with Route53 DNS records

This project is forked from https://github.com/calebdoxsey/kubernetes-cloudflare-sync

# Install
## Configure DNS settings to sync
Download the configuration templates in your work directory.

```shell
curl -LO https://github.com/jlandowner/kubernetes-route53-sync/releases/download/v1.2.0/kubernetes-route53-sync.tar.gz
tar -zxvf kubernetes-route53-sync.tar.gz 
```

Then update DNS name to sync and the other settings in `kubernetes/common/deployment.yaml`

```yaml
        env:
          - name: DNS_NAME
            value: "example.com"
```

For the options details, See the following [Available variable environments](https://github.com/jlandowner/kubernetes-route53-sync#available-variable-environments).

## Create AWS IAM Policy

```shell
aws iam create-policy --policy-name kubernetes-route53-sync --policy-document file://policy.json
```

The Output PolicyArn will be used later.

## Configure AWS IAM Credentials and Deploy

There are 2 ways to configure credentials. Choose one of them for your environment.

- Using Access Key
- Using IRSA (IAM Role for ServiceAccount) for EKS

### Using Access Key

1. Create IAM User

Replace YOUR_ACCOUNT_ID to yours.

```shell
aws iam create-user --user-name kubernetes-route53-sync
aws iam attach-user-policy --user-name kubernetes-route53-sync --policy-arn arn:aws:iam::YOUR_ACCOUNT_ID:policy/kubernetes-route53-sync
aws iam create-access-key --user-name kubernetes-route53-sync
```

Then replace <YOUR_AWS_ACCESS_KEY_ID> and <YOUR_AWS_SECRET_ACCESS_KEY> in `kubernetes/accesskey/kustomization.yaml`

2. Deploy

```shell
kustomize build kubernetes/accesskey | kubectl apply -f -
```

### Using IRSA (IAM Role for ServiceAccount) for EKS

You can also use [IRSA](https://docs.aws.amazon.com/ja_jp/eks/latest/userguide/iam-roles-for-service-accounts.html) if you run it on EKS.

Replace YOUR_EKS_CLUSTER_NAME and YOUR_ACCOUNT_ID to yours.

1. Create OIDC ID Provider

>Note: See the official docs if you do not use eksctl.
 https://docs.aws.amazon.com/ja_jp/eks/latest/userguide/enable-iam-roles-for-service-accounts.html

```shell
eksctl utils associate-iam-oidc-provider --cluster YOUR_EKS_CLUSTER_NAME --approve
```

2. Create IAM Role

>Note: See the official docs if you do not use eksctl.
 https://docs.aws.amazon.com/ja_jp/eks/latest/userguide/create-service-account-iam-policy-and-role.html

```shell
eksctl create iamserviceaccount \
    --name kubernetes-route53-sync \
    --namespace kube-system \
    --cluster YOUR_EKS_CLUSTER_NAME \
    --attach-policy-arn arn:aws:iam::YOUR_ACCOUNT_ID:policy/kubernetes-route53-sync \
    --approve \
    --override-existing-serviceaccounts
```

3. Configure ServiceAccount to use IAM Role

Replace YOUR_ACCOUNT_ID in `kubernetes/irsa/kustomization.yaml`

```yaml
commonAnnotations:
  eks.amazonaws.com/role-arn: arn:aws:iam::YOUR_ACCOUNT_ID:role/kubernetes-route53-sync
```

4. Deploy

```shell
kustomize build kubernetes/irsa | kubectl apply -f -
```

# Available variable environments
|name|description|example value|required|
|:--|:--|:--|:--|
|DNS_NAME|Route53 A Record to sync. Find Hostedzone ID by its sufix. |'k8s.example.com' (A Record in Hostedzone named "example.com")|true|
|DNS_TTL|Route53 Record TTL (default 300s)|'60'|false|
|HOSTEDZONE_ID|Specify Route53 Hostedzone ID especially when you have the subdomain at another hostedzone from root (default auto find by DNS_NAME suffix)|'XXXXXXXXXXXXX'|false|
|USE_INTERNAL_IP|Use Node Internal IP (default External IP)|'1'|false|
|ENABLE_DNS_ACCESS|Access to DNS for the reconciliation from the Pods (default 0)|'1'|false|
|NODE_SELECTOR|node selector query|'disktype=ssd' (default non)|false|
|HTTPS_PROXY|use proxy (protocol://host:port)|'http://your-proxy:1080'|false|
|NO_PROXY|not use proxy for specific endpoints|'sts.amazonaws.com'|false|

# LICENSE
MIT License
