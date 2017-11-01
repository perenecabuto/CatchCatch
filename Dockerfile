FROM scratch

ADD web/ /app/web/
ADD protobuf /app/web/protobuf

COPY catchcatch-server/catchcatch-server /app/server/

WORKDIR /app/server/

CMD ["./catchcatch-server"]
