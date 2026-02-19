FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /claudebot-mcp ./cmd/claudebot-mcp

FROM alpine:3.21
RUN apk add --no-cache ca-certificates \
    && addgroup -S app && adduser -S app -G app
COPY --from=build /claudebot-mcp /usr/local/bin/claudebot-mcp
USER app
EXPOSE 8080
ENTRYPOINT ["claudebot-mcp"]
