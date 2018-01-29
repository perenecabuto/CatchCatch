FROM scratch

ADD web/ /app/web/
ADD protobuf /app/web/protobuf

COPY server/server /app/server/

WORKDIR /app/server/

CMD ["./server"]
