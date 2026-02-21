FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=${VERSION}" -o /lorecraft ./cmd/lorecraft

FROM gcr.io/distroless/static-debian12

COPY --from=builder /lorecraft /lorecraft

ENTRYPOINT ["/lorecraft"]