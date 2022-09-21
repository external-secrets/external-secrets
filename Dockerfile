# provided by docker buildx
ARG TARGETOS
ARG TARGETARCH

ARG BUILDIMAGE=golang:1.19-alpine
ARG RUNIMAGE=gcr.io/distroless/static

FROM $BUILDIMAGE as build
ENV GOOS=$TARGETOS
ENV GOARCH=$TARGETARCH
ENV CGO_ENABLED=0
RUN apk update && apk add make

WORKDIR /work
COPY go.mod go.sum /work
RUN --mount=type=cache,target=/go/pkg/mod,sharing=private \
    go mod download
COPY . /work
RUN --mount=type=cache,target=/root/.cache/go-build,sharing=private \
    go build -o ./external-secrets main.go
RUN go tool nm ./external-secrets | grep FIPS

FROM $RUNIMAGE
COPY --from=build /work/external-secrets /bin/external-secrets

# Run as UID for nobody
USER 65534

ENTRYPOINT ["/bin/external-secrets"]
