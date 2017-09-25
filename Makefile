PLUGIN_NAME=nflex/ipam-driver
RELEASE=0.0.2

.PHONY: clean
clean:
	docker plugin ls --format "{{.Name}}" | grep ipam-driver | xargs -I {} docker plugin rm -f {} || true
	rm -rf ./build/rootfs

.PHONY: build
build:
	mkdir -p build/rootfs
	CGO_ENABLED=0 go build -ldflags='-s -w' -o build/rootfs/ipam-plugin main.go

.PHONY: create
create: clean build
	docker plugin create ${PLUGIN_NAME}:${RELEASE} ./build
	docker plugin create ${PLUGIN_NAME} ./build

.PHONY: enable
enable:
	docker plugin enable ${PLUGIN_NAME}

# You need to use docker login in order to push this to the public registry
.PHONY: push
push: create
	docker plugin push ${PLUGIN_NAME}:${RELEASE}
	docker plugin push ${PLUGIN_NAME}

.PHONY: install
install:
	docker plugin install ${PLUGIN_NAME} --disable --grant-all-permissions

.PHONY: pull
pull:
	docker plugin disable ${PLUGIN_NAME} || true
	docker plugin upgrade ${PLUGIN_NAME} --grant-all-permissions
