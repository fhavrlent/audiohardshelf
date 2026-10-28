FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
ARG TARGETOS TARGETARCH
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w -X main.version=$VERSION" -o /audiohardshelf .
RUN mkdir /data

# distroless/static has CA certificates and time zone data, and runs as a non-root user.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /audiohardshelf /usr/local/bin/audiohardshelf
# Sync state is kept here; 65532 is the nonroot user.
COPY --from=build --chown=65532:65532 /data /data
VOLUME /data

ENTRYPOINT ["/usr/local/bin/audiohardshelf"]
