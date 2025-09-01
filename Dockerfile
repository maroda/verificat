FROM golang:1.25-alpine3.22
LABEL app="verificat"
LABEL version="0.0.1"
LABEL org.opencontainers.image.source="https://github.com/maroda/verificat"
EXPOSE 4330
WORKDIR /go/src/verificat/
COPY . .
RUN go mod tidy
RUN go build -o /bin/verificat
ENTRYPOINT ["/bin/verificat"]
