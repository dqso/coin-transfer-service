FROM golang:1.27-alpine3.24 AS build
RUN CGO_ENABLED=0 go install -trimpath -ldflags="-s -w" github.com/jackc/tern/v2@v2.4.3

FROM alpine:3.24
RUN adduser -D -u 10001 app
USER app
COPY --from=build /go/bin/tern /usr/local/bin/tern
COPY migrations /migrations
ENTRYPOINT ["tern", "migrate", "--migrations", "/migrations"]