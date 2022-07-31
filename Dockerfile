ARG BASE_IMAGE=docker.io/library/ubuntu:22.04

### Base
FROM $BASE_IMAGE as base

USER root

ENV DEBIAN_FRONTEND=noninteractive

RUN groupadd -g 999 autopilot
RUN    useradd -r -u 999 -g autopilot autopilot
RUN    mkdir -p /home/autopilot
RUN    chown autopilot:0 /home/autopilot
RUN    chmod g=u /home/autopilot
RUN    apt-get update
RUN    apt-get install -y git git-lfs tini gpg tzdata
RUN    apt-get clean
RUN    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

ENV USER=autopilot

USER 999

WORKDIR /home/autopilot

### Build
FROM docker.io/library/golang:1.17 as build

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
