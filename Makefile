PLUGIN_NAME=nflex/ipam-driver
RELEASE=0.0.1

.PHONY: clean-plugin
clean-plugin:
	rm -rf ./build ./bin
	docker plugin disable ${PLUGIN_NAME}:${RELEASE} || true
	docker plugin rm ${PLUGIN_NAME}:${RELEASE} || true
	docker rm -vf tmp || true
	docker rmi ipam-build-image || true
	docker rmi ${PLUGIN_NAME}:rootfs || true

.PHONY: build-binary
build-binary:
	 docker build -t ipam-build-image -f Dockerfile.build .
	 docker create --name build-container ipam-build-image
	 docker cp build-container:/go/src/ipam-driver/bin .
	 docker rm -vf build-container
	 docker rmi ipam-build-image
	#go build -o bin/ipam-plugin

.PHONY: build-plugin
build-plugin: build-binary
	docker build -t ${PLUGIN_NAME}:rootfs .
	mkdir -p ./build/rootfs
	docker create --name tmp ${PLUGIN_NAME}:rootfs
	docker export tmp | tar -x -C ./build/rootfs
	cp config.json ./build/
	docker rm -vf tmp

.PHONY: create-plugin
create-plugin: clean-plugin build-plugin
	docker plugin create ${PLUGIN_NAME}:${RELEASE} ./build

.PHONY: enable-plugin
enable-plugin: create-plugin
	docker plugin enable ${PLUGIN_NAME}:${RELEASE}

.PHONY: push-plugin
push-plugin:  create-plugin
	docker plugin push ${PLUGIN_NAME}:${RELEASE}
