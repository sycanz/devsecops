# Stage 1: Builder
FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY app/go.mod app/go.sum ./
RUN go mod download

COPY app/ .

# Build the binary statically linked for Linux
RUN CGO_ENABLED=0 GOOS=linux go build -o /main ./cmd/server

# Stage 2: Final image
FROM gcr.io/distroless/static-debian13:nonroot

COPY --from=builder /main /main

USER nonroot:nonroot

EXPOSE 8000

ENTRYPOINT ["/main"]
