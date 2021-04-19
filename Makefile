BUILD_DIR ?= ./out

.PHONY: build
build: $(BUILD_DIR)
	go build \
	   -ldflags="-w -s" \
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
