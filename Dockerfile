FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /out/api ./cmd/api
RUN CGO_ENABLED=0 go build -o /out/worker ./cmd/worker

FROM alpine:3.21 AS runtime

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

RUN addgroup -S flowra && adduser -S -G flowra -u 10001 flowra

COPY --from=builder --chown=flowra:flowra /out/api /app/api
COPY --from=builder --chown=flowra:flowra /out/worker /app/worker
COPY --chown=flowra:flowra configs/config.yaml /etc/flowra/config.yaml

EXPOSE 8080
ENV FLOWRA_WORKER=true

USER flowra:flowra

ENTRYPOINT ["/app/api"]
