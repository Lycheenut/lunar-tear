FROM golang:1.26.10-alpine AS builder

ENV GOPROXY=https://goproxy.cn \
    GOSUMDB=sum.golang.org

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories

WORKDIR /usr/local/src
COPY . .

RUN apk add --no-cache \
    protobuf \
    protobuf-dev \
    protoc \
    make \
    libcap

RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest &&\
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest &&\
    go install github.com/pressly/goose/v3/cmd/goose@latest &&\
    PATH="$PATH:$(go env GOPATH)/bin" make proto &&\
    go build -o lunar-tear ./cmd/lunar-tear &&\
    go build -o export-snapshot ./cmd/export-snapshot &&\
    setcap cap_net_bind_service=+ep ./lunar-tear

FROM alpine:latest

WORKDIR /opt/lunar-tear

RUN chown 1000:1000 /opt/lunar-tear

USER 1000

COPY --from=builder /usr/local/src/lunar-tear .
COPY --from=builder /usr/local/src/export-snapshot .
COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY --from=builder /usr/local/src/migrations ./migrations

COPY entrypoint.sh .

ENTRYPOINT /opt/lunar-tear/entrypoint.sh
