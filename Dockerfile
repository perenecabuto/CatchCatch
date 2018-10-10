# https://medium.com/@chemidy/create-the-smallest-and-secured-golang-docker-image-based-on-scratch-4752223b7324
FROM golang:1.11.1-alpine as builder
COPY ./server $GOPATH/src/github.com/perenecabuto/CatchCatch/server
WORKDIR $GOPATH/src/github.com/perenecabuto/CatchCatch/server
# Install SSL ca certificates
RUN apk update && apk add ca-certificates
# Create appuser
RUN adduser -D -g '' appuser
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -tags netgo -o /go/bin/server

FROM scratch
ADD web/ /app/web/
ADD protobuf /app/web/protobuf
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /go/bin/server /app/server/
WORKDIR /app/server/
USER appuser
