FROM golang:1.15.8-alpine3.13 as base

RUN apk -U add --no-cache git ca-certificates && update-ca-certificates

RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid 10001 \
    autopilot

WORKDIR /go/src/github.com/argoproj/argocd-autopilot

COPY go.mod .
COPY go.sum .

RUN go mod download -x
RUN go mod verify

############################### CLI ###############################
### Compile
FROM golang:1.15.8-alpine3.13 as autopilot-build

WORKDIR /go/src/github.com/argoproj/argocd-autopilot

ARG OUT_DIR

RUN apk -U add --no-cache git make

COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY . .

ENV GOPATH ""
ENV GOBIN /go/bin

RUN make ./${OUT_DIR}/autopilot-linux-amd64

### Run
FROM alpine:3.13 as autopilot

WORKDIR /go/src/github.com/argoproj/argocd-autopilot

ARG OUT_DIR

# copy ca-certs and user details
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=base /etc/passwd /etc/passwd
COPY --from=base /etc/group /etc/group
COPY --chown=autopilot:autopilot --from=autopilot-build /go/src/github.com/argoproj/argocd-autopilot/${OUT_DIR}/autopilot-linux-amd64 /usr/local/bin/autopilot

USER autopilot:autopilot

ENTRYPOINT [ "autopilot" ]

############################### Agent ###############################
### Compile
FROM golang:1.15.8-alpine3.13 as gitops-agent-build

WORKDIR /go/src/github.com/argoproj/argocd-autopilot

ARG OUT_DIR

RUN apk -U add --no-cache git make

COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

COPY . .

ENV GOPATH ""
ENV GOBIN /go/bin

RUN make agent

### Run
FROM alpine:3.13 as gitops-agent

WORKDIR /go/src/github.com/argoproj/argocd-autopilot

ARG OUT_DIR

COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=base /etc/passwd /etc/passwd
COPY --from=base /etc/group /etc/group
COPY --chown=autopilot:autopilot --from=gitops-agent-build /go/src/github.com/argoproj/argocd-autopilot/${OUT_DIR}/gitops-agent /usr/local/bin/gitops-agent
COPY --chown=autopilot:autopilot --from=gitops-agent-build /go/src/github.com/argoproj/argocd-autopilot/server/assets /go/src/github.com/argoproj/argocd-autopilot/server/assets

USER autopilot:autopilot

ENTRYPOINT [ "gitops-agent" ]

CMD [ "start" ]
