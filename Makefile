# Makefile for automatic-cache-object-storage project

# Directories
BUILD_DIR := build
SRC_DIR := src

# Go commands
GO_GENERATE := go generate ./...
GO_BUILD := go build -o $(BUILD_DIR)/automatic-cache-object-storage

# Targets
.PHONY: all generate build clean rebuild

all: $(BUILD_DIR)/automatic-cache-object-storage

$(BUILD_DIR)/automatic-cache-object-storage: generate
	@mkdir -p $(BUILD_DIR)
	$(GO_BUILD)

generate:
	$(GO_GENERATE)

build:
	@mkdir -p $(BUILD_DIR)
	$(GO_BUILD)

clean:
	@rm -rf $(BUILD_DIR)

rebuild: clean build

run:
	$(GO_GENERATE)
	@mkdir -p $(BUILD_DIR)
	$(GO_BUILD)
	sudo ./$(BUILD_DIR)/automatic-cache-object-storage