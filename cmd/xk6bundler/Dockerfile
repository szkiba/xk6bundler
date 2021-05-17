FROM alpine
RUN apk add --no-cache ca-certificates && \
    adduser -D -u 12345 -g 12345 k6
COPY {{.Output}} /usr/bin/k6

USER 12345
WORKDIR /home/k6
ENTRYPOINT ["k6"]
