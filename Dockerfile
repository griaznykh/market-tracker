ARG BUILD_IMAGE=golang:1.26
ARG RELEASE_IMAGE=alpine

# builder
FROM --platform=$BUILDPLATFORM ${BUILD_IMAGE} AS builder

ARG VERSION

WORKDIR /src
COPY . .

ARG TARGETOS TARGETARCH
ENV GOOS=$TARGETOS
ENV GOARCH=$TARGETARCH
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg \
    CGO_ENABLED=0 go build -mod vendor -o /marketservice \
    -ldflags "-s -w -X main.Version=${VERSION}"

# clean image for end users
FROM ${RELEASE_IMAGE}

ARG TARGETOS TARGETARCH

# define home dir, user and group for running the app
ENV APP_HOME=/home/marketservice \
    APP_UID=2100 \
    APP_GID=2100 \
    APP_USER=marketservice \
    APP_GROUP=marketservice

RUN set -xe &&  \
    addgroup -g ${APP_GID} -S ${APP_GROUP} && \
    adduser -S -s /usr/sbin/nologin \
       -h ${APP_HOME} -D \
       -u ${APP_UID} -G ${APP_GROUP} \
       ${APP_USER} && \
    chown -R ${APP_USER}:${APP_GROUP} ${APP_HOME} && \
    chmod -R 0775 ${APP_HOME}

COPY --from=builder --chown=${APP_USER} /marketservice ./marketservice
RUN chmod +x ./marketservice

EXPOSE 8000

USER ${APP_USER}

ENTRYPOINT ["./marketservice"]