FROM golang:1.17-alpine as builder
WORKDIR /go/src/github.com/pelletier/go-toml
COPY . .
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
RUN go install ./...

FROM scratch
COPY --from=builder /go/bin/tomll /usr/bin/tomll
COPY --from=builder /go/bin/tomljson /usr/bin/tomljson
COPY --from=builder /go/bin/jsontoml /usr/bin/jsontoml
