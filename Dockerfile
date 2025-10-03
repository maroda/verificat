FROM alpine:latest
LABEL app=verificat
LABEL org.opencontainers.image.source=https://github.com/maroda/verificat
WORKDIR /
COPY verificat .
ENTRYPOINT ["/verificat"]
