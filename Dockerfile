FROM golang:1.25-alpine3.22
LABEL app=verificat
LABEL org.opencontainers.image.source=https://github.com/maroda/verificat
ENTRYPOINT ["/verificat"]
COPY verificat /
