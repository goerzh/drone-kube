FROM golang:1.11.1-alpine as builder

#RUN apk update && \
#    apk add ca-certificates && \
#    rm -rf /var/cache/apk/*

COPY . /go/src/github.com/goerzh/drone-kube
WORKDIR /go/src/github.com/goerzh/drone-kube

RUN CGO_ENABLED=0 go build -o bin/drone-kube github.com/goerzh/drone-kube/cmd

FROM alpine:3.5

COPY --from=builder /go/src/github.com/goerzh/drone-kube/bin/drone-kube /bin/drone-kube

CMD ["/bin/drone-kube"]
