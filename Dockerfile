FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api

FROM alpine:3.21 AS runtime

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /out/api /app/api
COPY configs/config.yaml /etc/flowra/config.yaml

EXPOSE 8080
ENV FLOWRA_WORKER=true

ENTRYPOINT ["/app/api"]
