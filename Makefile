NAME=kubernetes-route53-sync
PHONY: release-asset
release-asset:
	tar -zcvf $(NAME).tar.gz kubernetes LICENSE README.md policy.json