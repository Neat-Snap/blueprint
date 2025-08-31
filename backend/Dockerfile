FROM golang:1.24.4 AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN touch .env

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /server ./

FROM gcr.io/distroless/base-debian12
WORKDIR /

USER 65532:65532

COPY --from=builder /server /server
COPY --from=builder /app/.env /.env

EXPOSE 8080

ENTRYPOINT ["/server"]
