ARG BASE_DEV_CONTAINER=ubuntu:22.04
ARG BASE_RUN_CONTAINER=ubuntu:22.04
ARG GO_VERSION=1.24.3
ARG ARCH
ARG OS

# Setup build container
FROM ${BASE_DEV_CONTAINER} AS build
ARG GO_VERSION
ARG ARCH
ARG OS

RUN apt-get -y update \
 && apt-get -y install ca-certificates build-essential libgomp1 cmake git pkg-config nasm curl \
 && apt-get -y install libfreetype6-dev libmp3lame-dev libopus-dev libvorbis-dev libvpx-dev libx264-dev libdav1d-dev

# Install go
RUN curl -sL https://golang.org/dl/go${GO_VERSION}.${OS}-${ARCH}.tar.gz | tar -C /usr/local -xz
ENV PATH=$PATH:/usr/local/go/bin

# Copy source
WORKDIR /app
COPY . .

# Make whisper
RUN make -j$(nproc)

# Setup runtime container
FROM ${BASE_RUN_CONTAINER} AS runtime
RUN apt-get -y update \
 && apt-get -y install ca-certificates libgomp1 \
 && apt-get -y install libfreetype6 libmp3lame0 libopus0 libvorbis0a libvpx-dev x264 dav1d

# Copy built whisper binary and entrypoint script
COPY --from=build --chmod=755 /app/build/whisper /usr/local/bin
COPY --chmod=755 etc/entrypoint.sh .

# Container environment
ENV WHISPER_DIR="/data"
VOLUME [ "/data" ]
ENTRYPOINT [ "/entrypoint.sh" ]
STOPSIGNAL SIGQUIT
EXPOSE 80
