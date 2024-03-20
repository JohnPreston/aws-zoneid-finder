ARG GO_VERSION=1.21.3
ARG GO_IMAGE=public.ecr.aws/docker/library/golang:${GO_VERSION:-latest}
ARG ALPINE_IMAGE=public.ecr.aws/docker/library/alpine:latest

FROM $GO_IMAGE as builder
WORKDIR /app
COPY main.go /app/main.go
COPY go.mod /app/go.mod
RUN go get aws-zoneid-finder
RUN go build

FROM $ALPINE_IMAGE as app
WORKDIR /app
COPY --from=builder /app/aws-zoneid-finder /app/aws-zoneid-finder
COPY entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/aws-zoneid-finder /app/entrypoint.sh
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2
ENTRYPOINT ["/app/entrypoint.sh"]
