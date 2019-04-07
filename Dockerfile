FROM golang:1.12-alpine3.9 as builder
WORKDIR /go/src/github.com/pelletier/go-toml
COPY . .
RUN go build && \
      cd cmd/tomll && \
      go build && \
      cd ../tomljson && \
      go build

FROM alpine:3.9
COPY --from=builder /go/src/github.com/pelletier/go-toml/cmd/tomll/tomll /usr/bin/tomll
COPY --from=builder /go/src/github.com/pelletier/go-toml/cmd/tomljson/tomljson /usr/bin/tomljson
RUN chmod +x /usr/bin/tomll /usr/bin/tomljson
