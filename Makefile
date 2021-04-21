BUILD_DIR ?= ./out

VERSION := $(shell git describe --tags)
COMMIT_NO := $(shell git rev-parse HEAD 2> /dev/null || true)
COMMIT ?= $(if $(shell git status --porcelain --untracked-files=no),"${COMMIT_NO}-dirty","${COMMIT_NO}")

LDFLAGS := -X github.com/rancher-sandbox/docker-machine-driver-hyperkit/pkg/hyperkit.version=$(VERSION) \
           -X github.com/rancher-sandbox/docker-machine-driver-hyperkit/pkg/hyperkit.gitCommitID=$(COMMIT)


.PHONY: build
build: $(BUILD_DIR)
	go build \
	   -ldflags="-w -s $(LDFLAGS)" \
	   -o $(BUILD_DIR)/docker-machine-driver-hyperkit \
	   main.go
	sudo chown root:wheel $(BUILD_DIR)/docker-machine-driver-hyperkit
	sudo chmod u+s $(BUILD_DIR)/docker-machine-driver-hyperkit
	sudo -k

.PHONY: clean
clean:
	rm -rf $(BUILD_DIR)

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)
