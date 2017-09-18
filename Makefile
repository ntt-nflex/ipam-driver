PLUGIN_NAME=nflex/ipam-driver
RELEASE=0.0.1

.PHONY: clean
clean:
	rm -rf ./build/rootfs
	docker plugin disable ${PLUGIN_NAME}:${RELEASE} || true
	docker plugin rm ${PLUGIN_NAME}:${RELEASE} || true

.PHONY: build
build:
	mkdir -p build/rootfs
	CGO_ENABLED=0 go build -ldflags='-s -w' -o build/rootfs/ipam-plugin main.go

.PHONY: create
create: clean build
	docker plugin create ${PLUGIN_NAME}:${RELEASE} ./build

.PHONY: enable
enable: create
	docker plugin enable ${PLUGIN_NAME}:${RELEASE}

.PHONY: push
push: create
	docker plugin push ${PLUGIN_NAME}:${RELEASE}
