# Paths to packages
DOCKER=$(shell which docker)
GIT=$(shell which git)
GO=$(shell which go)
CMAKE=$(shell which cmake)

# Set OS and Architecture
ARCH ?= $(shell arch | tr A-Z a-z | sed 's/x86_64/amd64/' | sed 's/i386/amd64/' | sed 's/armv7l/arm/' | sed 's/aarch64/arm64/')
OS ?= $(shell uname | tr A-Z a-z)
VERSION ?= $(shell git describe --tags --always | sed 's/^v//')
DOCKER_REGISTRY ?= ghcr.io/mutablelogic

# Set docker tag, etc
BUILD_TAG := ${DOCKER_REGISTRY}/go-whisper-${OS}-${ARCH}:${VERSION}
ROOT_PATH := $(CURDIR)
BUILD_DIR := build

# Build flags
BUILD_MODULE := $(shell cat go.mod | head -1 | cut -d ' ' -f 2)
BUILD_LD_FLAGS += -X $(BUILD_MODULE)/pkg/version.GitSource=${BUILD_MODULE}
BUILD_LD_FLAGS += -X $(BUILD_MODULE)/pkg/version.GitTag=$(shell git describe --tags --always)
BUILD_LD_FLAGS += -X $(BUILD_MODULE)/pkg/version.GitBranch=$(shell git name-rev HEAD --name-only --always)
BUILD_LD_FLAGS += -X $(BUILD_MODULE)/pkg/version.GitHash=$(shell git rev-parse HEAD)
BUILD_LD_FLAGS += -X $(BUILD_MODULE)/pkg/version.GoBuildTime=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
BUILD_FLAGS = -ldflags "-s -w $(BUILD_LD_FLAGS)" 
TEST_FLAGS = -v

# If GGML_CUDA is set, then add a cuda tag for the go ${BUILD FLAGS}
ifeq ($(GGML_CUDA),1)
	TEST_FLAGS += -tags cuda
	BUILD_FLAGS += -tags cuda
	CUDA_DOCKER_ARCH ?= all
endif

# Targets
all: whisper api

# Generate the pkg-config files
generate: mkdir go-tidy
	@echo "Generating pkg-config"
	@PKG_CONFIG_PATH=${ROOT_PATH}/${BUILD_DIR} go generate ./sys/whisper

# Make whisper
whisper: mkdir generate go-tidy libwhisper
	@echo "Building whisper"
	echo "PKG_CONFIG_PATH=${ROOT_PATH}/${BUILD_DIR} ${GO} build ${BUILD_FLAGS} -o ${BUILD_DIR}/whisper ./cmd/whisper"
	@PKG_CONFIG_PATH=${ROOT_PATH}/${BUILD_DIR} ${GO} build ${BUILD_FLAGS} -o ${BUILD_DIR}/whisper ./cmd/whisper

# Make api
api: mkdir go-tidy
	@echo "Building api"
	@${GO} build ${BUILD_FLAGS} -o ${BUILD_DIR}/api ./cmd/api

# Build docker container
docker: docker-dep submodule
	@echo build docker image: ${BUILD_TAG} for ${OS}/${ARCH}
	@${DOCKER} build \
		--tag ${BUILD_TAG} \
		--build-arg ARCH=${ARCH} \
		--build-arg OS=${OS} \
		--build-arg SOURCE=${BUILD_MODULE} \
		--build-arg VERSION=${VERSION} \
		--build-arg GGML_CUDA=${GGML_CUDA} \
		-f etc/Dockerfile .

# Test whisper bindings
test: generate libwhisper
	@echo "Running tests (sys)"
	@PKG_CONFIG_PATH=${ROOT_PATH}/${BUILD_DIR} ${GO} test ${TEST_FLAGS} ./sys/whisper/...
	@echo "Running tests (pkg)"
	@PKG_CONFIG_PATH=${ROOT_PATH}/${BUILD_DIR} ${GO} test ${TEST_FLAGS} ./pkg/...
	@echo "Running tests (whisper)"
	@PKG_CONFIG_PATH=${ROOT_PATH}/${BUILD_DIR} ${GO} test ${TEST_FLAGS} ./

libwhisper: mkdir submodule cmake-dep 
	@${CMAKE} -S third_party/whisper.cpp -B ${BUILD_DIR} -DBUILD_SHARED_LIBS=0 -DGGML_CUDA=${GGML_CUDA}
	@${CMAKE} --build ${BUILD_DIR} -j --config Release

# Push docker container
docker-push: docker-dep 
	@echo push docker image: ${BUILD_TAG}
	@${DOCKER} push ${BUILD_TAG}

# Update submodule to the latest version
submodule-update: git-dep
	@echo "Updating submodules"
	@${GIT} submodule foreach git pull origin master

# Submodule checkout
submodule: git-dep
	@echo "Checking out submodules"
	@${GIT} submodule update --init --recursive --remote

# Submodule clean
submodule-clean: git-dep
	@echo "Cleaning submodules"
	@${GIT} reset --hard
	@${GIT} submodule sync --recursive
	@${GIT} submodule update --init --force --recursive
	@${GIT} clean -ffdx
	@${GIT} submodule foreach --recursive git clean -ffdx	

# Check for docker
docker-dep:
	@test -f "${DOCKER}" && test -x "${DOCKER}"  || (echo "Missing docker binary" && exit 1)

# Check for docker
cmake-dep:
	@test -f "${CMAKE}" && test -x "${CMAKE}"  || (echo "Missing cmake binary" && exit 1)

# Check for git
git-dep:
	@test -f "${GIT}" && test -x "${GIT}"  || (echo "Missing git binary" && exit 1)

# Check for go
go-dep:
	@test -f "${GO}" && test -x "${GO}"  || (echo "Missing go binary" && exit 1)

# Make build directory
mkdir:
	@echo Mkdir ${BUILD_DIR}
	@install -d ${BUILD_DIR}

# go mod tidy
go-tidy: go-dep
	@echo Tidy
	@${GO} mod tidy

# Clean
clean: submodule-clean go-tidy
	@echo "Cleaning"
	@rm -rf ${BUILD_DIR}
