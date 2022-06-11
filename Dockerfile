FROM golang:1.18 AS builder
COPY go.mod /root/go/
COPY go.sum /root/go/
COPY metrix.go /root/go/
WORKDIR /root/go
RUN go mod tidy && \
    go build metrix.go

FROM gcr.io/distroless/base-debian11
COPY --from=builder /root/go/metrix /
ENV SSL_CERT_DIR=/var/run/secrets/kubernetes.io/serviceaccount/
ENV SSL_CERT_FILE=ca.crt
USER 1001