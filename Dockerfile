FROM gcr.io/distroless/static-debian10
LABEL org.opencontainers.image.source https://github.com/stern/stern
COPY stern /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/stern"]
