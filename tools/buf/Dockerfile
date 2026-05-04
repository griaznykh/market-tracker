FROM debian:trixie-slim

# install additional tools
RUN apt-get update && apt-get install -y \
    curl \
    ca-certificates \
    unzip \
    --no-install-recommends

# install go
COPY --from=golang:1.26.2-alpine /usr/local/go /usr/local/go
ENV PATH="${PATH}:/usr/local/go/bin"
RUN go version

ENV GOPATH /go
ENV PATH $GOPATH/bin:$PATH
RUN set -xe && \
    mkdir -p "$GOPATH/src" "$GOPATH/bin" && \
    chmod -R 777 "$GOPATH"

ENV GO111MODULE="on"

# install go plugins
RUN set -xe && \
    go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.7 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1 && \
    go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.27.1 && \
    go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.27.1 && \
    go install github.com/bufbuild/buf/cmd/buf@v1.56.0

# setup workspace
RUN mkdir -p /workspace

WORKDIR /workspace

ENTRYPOINT ["buf"]
