FROM golang:1.27-alpine3.24 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/coin-transfer ./app/cmd/main.go

FROM alpine:3.24
RUN adduser -D -u 10001 app
USER app
COPY --from=build /out/coin-transfer /coin-transfer
EXPOSE 3000
ENTRYPOINT ["/coin-transfer"]
