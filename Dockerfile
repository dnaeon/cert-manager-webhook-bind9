FROM golang:1.21-alpine AS builder

WORKDIR /workspace

COPY . .

RUN CGO_ENABLED=0 go build -v -o webhook

FROM alpine:3.18.4

RUN apk add --no-cache ca-certificates bash bind-tools

COPY --from=builder /workspace/webhook /usr/local/bin/webhook
COPY --from=builder /workspace/scripts/acme-challenge-helper.sh /usr/local/bin

ENTRYPOINT ["webhook"]
