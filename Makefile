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
DOCKER_FILE ?= etc/Dockerfile

# Set docker tag, etc
BUILD_TAG := ${DOCKER_REGISTRY}/go-whisper-${OS}-${ARCH}:${VERSION}
ROOT_PATH := $(CURDIR)
BUILD_DIR ?= "build"
PREFIX ?= ${BUILD_DIR}/install

# Build flags
BUILD_MODULE := $(shell cat go.mod | head -1 | cut -d ' ' -f 2)
BUILD_LD_FLAGS += -X $(BUILD_MODULE)/pkg/version.GitSource=${BUILD_MODULE}
BUILD_LD_FLAGS += -X $(BUILD_MODULE)/pkg/version.GitTag=$(shell git describe --tags --always)
BUILD_LD_FLAGS += -X $(BUILD_MODULE)/pkg/version.GitBranch=$(shell git name-rev HEAD --name-only --always)
BUILD_LD_FLAGS += -X $(BUILD_MODULE)/pkg/version.GitHash=$(shell git rev-parse HEAD)
BUILD_LD_FLAGS += -X $(BUILD_MODULE)/pkg/version.GoBuildTime=$(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
BUILD_FLAGS = -ldflags "-s -w $(BUILD_LD_FLAGS)" 
TEST_FLAGS = -v
CMAKE_FLAGS = -DBUILD_SHARED_LIBS=OFF 

# If GGML_CUDA is set, then add a cuda tag for the go ${BUILD FLAGS}
ifeq ($(GGML_CUDA),1)
	TEST_FLAGS += -tags cuda
	BUILD_FLAGS += -tags cuda
	CUDA_DOCKER_ARCH ?= all
	CMAKE_FLAGS += -DGGML_CUDA=ON
	BUILD_TAG := "${BUILD_TAG}-cuda"
	DOCKER_FILE = etc/Dockerfile.cuda-test
endif

# If GGML_VULKAN is set, then add a vulkan tag for the go ${BUILD FLAGS}
ifeq ($(GGML_VULKAN),1)
	TEST_FLAGS += -tags vulkan
	BUILD_FLAGS += -tags vulkan
	CMAKE_FLAGS += -DGGML_VULKAN=ON
	BUILD_TAG := "${BUILD_TAG}-vulkan"
	DOCKER_FILE = etc/Dockerfile.vulkan
endif

# Targets
all: whisper

# Generate the pkg-config files
generate: mkdir go-tidy libwhisper
	@echo "Generating pkg-config"
	@mkdir -p ${BUILD_DIR}/lib/pkgconfig
	@PKG_CONFIG_PATH=$(shell realpath ${PREFIX})/lib/pkgconfig PREFIX="$(shell realpath ${PREFIX})" go generate ./sys/whisper

# Make whisper
whisper: generate libwhisper libffmpeg
	@echo "Building whisper"
	@PKG_CONFIG_PATH=$(shell realpath ${PREFIX})/lib/pkgconfig CGO_LDFLAGS_ALLOW="-(W|D).*" ${GO} build ${BUILD_FLAGS} -o ${BUILD_DIR}/whisper ./cmd/whisper

# Make api
api: mkdir go-tidy
	@echo "Building api"
	@${GO} build ${BUILD_FLAGS} -o ${BUILD_DIR}/api ./cmd/api

# Test whisper bindings
test: generate libwhisper
	@echo "Running tests (sys) with ${PREFIX}/lib"
	PKG_CONFIG_PATH=$(shell realpath ${PREFIX})/lib ${GO} test ${TEST_FLAGS} ./sys/whisper/...
	@echo "Running tests (pkg)"
	@PKG_CONFIG_PATH=$(shell realpath ${PREFIX})/lib ${GO} test ${TEST_FLAGS} ./pkg/...
	@echo "Running tests (whisper)"
	@PKG_CONFIG_PATH=$(shell realpath ${PREFIX})/lib ${GO} test ${TEST_FLAGS} ./

# make libwhisper and install at ${PREFIX}
libwhisper: mkdir submodule cmake-dep 
	@echo "Making libwhisper with ${CMAKE_FLAGS}"
	@${CMAKE} -S third_party/whisper.cpp -B ${BUILD_DIR} -DCMAKE_BUILD_TYPE=Release ${CMAKE_FLAGS}
	@${CMAKE} --build ${BUILD_DIR} -j --config Release
	@${CMAKE} --install ${BUILD_DIR} --prefix $(shell realpath ${PREFIX})

# make ffmpeg libraries and install at ${PREFIX}
libffmpeg: mkdir submodule
	@echo "Making ffmpeg libraries => ${PREFIX}"
	@mkdir -p ${BUILD_DIR}
	@mkdir -p ${PREFIX}
	@BUILD_DIR=$(shell realpath ${BUILD_DIR}) PREFIX=$(shell realpath ${PREFIX}) make -C third_party/go-media ffmpeg

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
		--build-arg GGML_VULKAN=${GGML_VULKAN} \
		-f ${DOCKER_FILE} .

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
	@echo Mkdir ${PREFIX}
	@install -d ${PREFIX}

# go mod tidy
go-tidy: go-dep
	@echo Tidy
	@${GO} mod tidy
	@${GO} clean -cache

# Clean
clean: submodule-clean go-tidy
	@echo "Cleaning"
	@rm -rf ${BUILD_DIR}
