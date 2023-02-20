ARG BASE_IMAGE=docker.io/library/ubuntu:22.04

### Base
FROM $BASE_IMAGE as base

USER root

ENV DEBIAN_FRONTEND=noninteractive

RUN groupadd -g 999 autopilot && \
    useradd -r -u 999 -g autopilot autopilot && \
    mkdir -p /home/autopilot && \
    chown autopilot:0 /home/autopilot && \
    chmod g=u /home/autopilot && \
    apt-get update -y && \
    apt-get upgrade -y && \
    apt-get install -y git git-lfs tini gpg tzdata && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

ENV USER=autopilot

USER 999

WORKDIR /home/autopilot

### Build
FROM docker.io/library/golang:1.19 as build

WORKDIR /go/src/github.com/argoproj-labs/argocd-autopilot

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN make local DEV_MODE=false

### Run
FROM base

COPY --from=build /go/src/github.com/argoproj-labs/argocd-autopilot/dist/* /usr/local/bin/argocd-autopilot

ENTRYPOINT [ "argocd-autopilot" ]
