# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

# Build source code from ./cmd/gateway to gateway folder in binary format
RUN CGO_ENABLED=0 GOOS=linux go build -o gateway ./cmd/gateway

# Runtime stage
FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app

COPY --from=builder /app/gateway .

EXPOSE 8080

USER nonroot:nonroot

CMD ["./gateway"]
