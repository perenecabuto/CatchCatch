FROM scratch

ADD web/ /app/web/
ADD protobuf /app/web/protobuf

COPY catchcatch-server/server /app/server/

WORKDIR /app/server/

CMD ["./server"]
