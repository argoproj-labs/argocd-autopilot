FROM golang:1.16.3-alpine3.13 as base

WORKDIR /go/src/github.com/argoproj/argocd-autopilot

RUN apk -U add --no-cache git ca-certificates && update-ca-certificates

RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid 10001 \
    autopilot


COPY go.mod .
COPY go.sum .

RUN go mod download
RUN go mod verify

############################### CLI ###############################
### Compile
FROM golang:1.16.3-alpine3.13 as autopilot-build

WORKDIR /go/src/github.com/argoproj/argocd-autopilot

ARG OUT_DIR

RUN apk -U add --no-cache git make

COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=base /go/pkg/mod /go/pkg/mod

COPY . .

ENV GOPATH /go
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
