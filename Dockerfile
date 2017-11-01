FROM alpine:latest

RUN mkdir -p /app/web
ADD web/ /app/web/
ADD protobuf /app/web/protobuf

RUN mkdir -p /app/server
COPY catchcatch-server/catchcatch-server /app/server/

WORKDIR /app/server/

CMD ["./catchcatch-server"]
