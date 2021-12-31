FROM golang:1.17.5-alpine3.15 as builder
WORKDIR /go/src/github.com/pelletier/go-toml
COPY . .
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
RUN go install ./...

FROM scratch
ENV PATH "$PATH:/bin"
COPY --from=builder /go/bin/linux_amd64/tomll /bin/tomll
COPY --from=builder /go/bin/linux_amd64/tomljson /bin/tomljson
COPY --from=builder /go/bin/linux_amd64/jsontoml /bin/jsontoml
