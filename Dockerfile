FROM golang:1.25-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/chem-platform .

FROM debian:bookworm-slim
WORKDIR /app
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata && rm -rf /var/lib/apt/lists/*
COPY --from=builder /out/chem-platform /app/chem-platform
COPY --from=builder /src/conf /app/conf
COPY --from=builder /src/template /app/template
COPY --from=builder /src/assets /app/assets
ENV TZ=Asia/Shanghai GIN_MODE=release
EXPOSE 8080
CMD ["/app/chem-platform", "httpd"]
