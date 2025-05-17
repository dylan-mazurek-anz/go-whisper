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
 && apt-get -y install software-properties-common build-essential cmake git curl \
 && add-apt-repository -y ppa:ubuntuhandbook1/ffmpeg6 \
 && apt-get -y update \
 && apt-get -y install libavformat-dev libavcodec-dev libavdevice-dev libavfilter-dev libavutil-dev libswscale-dev libswresample-dev

# Install go
RUN curl -sL https://golang.org/dl/go${GO_VERSION}.${OS}-${ARCH}.tar.gz | tar -C /usr/local -xz
ENV PATH=$PATH:/usr/local/go/bin

# Copy source
WORKDIR /app
COPY . .

# Make whisper-server
RUN make -j$(nproc)

# Setup runtime container
FROM ${BASE_RUN_CONTAINER} AS runtime
RUN apt-get -y update \
 && apt-get -y install software-properties-common \
 && add-apt-repository -y ppa:ubuntuhandbook1/ffmpeg6 \
 && apt-get -y update \
 && apt-get -y install libavformat60 libavcodec60 libavdevice60 libavfilter9 libavutil58 libswscale7 libswresample4 \
 && apt -y remove software-properties-common \
 && apt -y autoremove
COPY --from=build --chmod=755 /app/build/whisper /usr/local/bin/whisper
COPY --from=build --chmod=755 /app/build/api /usr/local/bin/api
COPY --chmod=755 etc/entrypoint.sh .

# Entrypoint when running the server
ENTRYPOINT [ "/entrypoint.sh" ]
STOPSIGNAL SIGQUIT
EXPOSE 80 443
VOLUME [ "/data" ]
CMD [ "/usr/local/bin/whisper", "server", "--dir", "/data", "--listen", ":80", "--endpoint", "/v1"  ]
