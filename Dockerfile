FROM golang:1.26-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -o /out/api main.go

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /out/api ./api
COPY --from=builder /src/docs ./docs
RUN mkdir -p ./etc/config ./etc/featureflag ./etc/vault

EXPOSE 12000

ENTRYPOINT ["./api", "conjure", "serve"]
