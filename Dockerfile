FROM golang:1.17.8-alpine3.15 as base

WORKDIR /go/src/github.com/argoproj-labs/argocd-autopilot

RUN apk -U add --no-cache git ca-certificates && update-ca-certificates

RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/home/autopilot" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid 10001 \
    autopilot

COPY go.mod .
COPY go.sum .

RUN go mod download -x
RUN go mod verify

############################### CLI ###############################
### Compile
FROM golang:1.17.8-alpine3.15 as autopilot-build

WORKDIR /go/src/github.com/argoproj-labs/argocd-autopilot

RUN apk -U add --no-cache git make

COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=base /go/pkg/mod /go/pkg/mod

COPY . .

ENV GOPATH /go
ENV GOBIN /go/bin

RUN make local DEV_MODE=false

### Run
FROM alpine:3.15 as autopilot

WORKDIR /home/autopilot

RUN apk -U add --no-cache git

# copy ca-certs and user details
COPY --from=base /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=base /etc/passwd /etc/passwd
COPY --from=base /etc/group /etc/group
COPY --chown=autopilot:autopilot --from=autopilot-build /go/src/github.com/argoproj-labs/argocd-autopilot/dist/* /usr/local/bin/argocd-autopilot

RUN chown -R autopilot:autopilot /home/autopilot
USER autopilot:autopilot

ENTRYPOINT [ "argocd-autopilot" ]
