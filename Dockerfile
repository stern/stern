FROM gcr.io/distroless/static-debian10
COPY stern /usr/local/bin/
ENTRYPOINT ["/usr/local/bin/stern"]
