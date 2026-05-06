# Stage 1: build
FROM golang:1.20-alpine AS build

RUN apk add --no-cache git ca-certificates

WORKDIR /app
# copy module files first for build cache (if present)
COPY src/go.mod src/go.sum ./
RUN if [ -f /app/go.mod ]; then go mod download; fi

# copy the rest of the source tree from src
COPY src ./

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /usr/local/bin/pollio .

# Stage 2: runtime
FROM alpine:3.18

RUN apk add --no-cache ca-certificates tzdata

# create non-root user and group
RUN addgroup -S pollio && adduser -S -G pollio pollio

# create log dir and set ownership
RUN mkdir -p /pollio/config && chown -R pollio:pollio /pollio/config && touch /pollio/config/services.yaml

# copy binary from builder
COPY --from=build /usr/local/bin/pollio /usr/local/bin/pollio

# copy entrypoint
COPY entrypoint.sh /usr/local/bin/entrypoint.sh

# default working dir and user
WORKDIR /home/pollio

ENV POLLIO_SERVICES_FILE="/pollio/config/services.yaml"

# ensure entrypoint runs with sh
ENTRYPOINT ["sh", "/usr/local/bin/entrypoint.sh"]

# run as non-root
USER pollio
