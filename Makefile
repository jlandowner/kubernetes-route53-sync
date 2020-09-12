NAME=kubernetes-route53-sync
PHONY: release
release:
	tar -zcvf $(NAME).tar.gz kubernetes LICENSE README.md policy.json