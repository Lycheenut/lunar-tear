FROM alpine:latest AS builder

ARG GO_VERSION=1.25.12
RUN apk add --no-cache ca-certificates curl tar && \
    case "$(apk --print-arch)" in \
      x86_64) GO_ARCH=amd64; GO_SHA256=234828b7a89e0e303d2556310ee549fbcf253d28de937bac3da13d6294262ac1 ;; \
      aarch64) GO_ARCH=arm64; GO_SHA256=8b5884aef89600aef5b0b051fb971f11f49bb996521e911f30f02a66884f7bd2 ;; \
      *) echo "unsupported architecture: $(apk --print-arch)" >&2; exit 1 ;; \
    esac && \
    curl --fail --location --retry 3 \
      "https://golang.google.cn/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz" \
      --output /tmp/go.tar.gz && \
    echo "${GO_SHA256}  /tmp/go.tar.gz" | sha256sum -c - && \
    tar -C /usr/local -xzf /tmp/go.tar.gz && \
    rm /tmp/go.tar.gz

ENV PATH="/usr/local/go/bin:${PATH}" \
    CGO_ENABLED=0 \
    GOTOOLCHAIN=local

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
    setcap cap_net_bind_service=+ep ./lunar-tear

FROM alpine:latest

WORKDIR /opt/lunar-tear

RUN chown 1000:1000 /opt/lunar-tear

USER 1000

COPY --from=builder /usr/local/src/lunar-tear .
COPY --from=builder /root/go/bin/goose /usr/local/bin/goose
COPY --from=builder /usr/local/src/migrations ./migrations

COPY entrypoint.sh .

ENTRYPOINT /opt/lunar-tear/entrypoint.sh
