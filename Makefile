NAME=kubernetes-route53-sync
VERSION=v1.2.0
PHONY: release-asset
PHONY: update-version

release-asset:
	tar -zcvf $(NAME).tar.gz kubernetes LICENSE README.md policy.json

update-version:
	sed -i.bk "s/`grep 'image:' kubernetes/common/deployment.yaml | awk '{print substr($$2, index($$2, "release-v")+8)}'`/$(VERSION)/" README.md
	sed -i.bk "s/`grep 'image:' kubernetes/common/deployment.yaml | awk '{print substr($$2, index($$2, "release-v")+8)}'`/$(VERSION)/" kubernetes/common/deployment.yaml

create-remote-tag:
	git tag $(VERSION)
	git push origin $(VERSION)

delete-remote-tag:
	git push --delete origin $(VERSION)
	git tag -d $(VERSION)