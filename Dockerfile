FROM scratch
LABEL org.opencontainers.image.source=https://github.com/maroda/verificat

# Install the ca-certificate package
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*
# Update the CA certificates in the container
RUN update-ca-certificates

ENTRYPOINT ["/verificat"]
COPY verificat /
