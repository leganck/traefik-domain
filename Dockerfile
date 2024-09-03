# build stage
FROM --platform=$BUILDPLATFORM golang:1.22.4-alpine AS builder

WORKDIR /app
COPY . .
ARG TARGETOS
ARG TARGETARCH

RUN apk add --no-cache git make tzdata \
    && GOOS=$TARGETOS GOARCH=$TARGETARCH make clean build

# final stage
FROM alpine
LABEL name=traefik-domain
LABEL url=https://github.com/leganck/traefik-domain
RUN apk add --no-cache curl grep
ENV TZ=Asia/Shanghai

WORKDIR /app
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /app/traefik-domain /app/traefik-domain
ENTRYPOINT ["/app/traefik-domain"]
