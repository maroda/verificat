FROM scratch
LABEL org.opencontainers.image.source=https://github.com/maroda/verificat
ENTRYPOINT ["/verificat"]
COPY verificat /
