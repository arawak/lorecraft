FROM gcr.io/distroless/static-debian12

COPY lorecraft /lorecraft

ENTRYPOINT ["/lorecraft"]
