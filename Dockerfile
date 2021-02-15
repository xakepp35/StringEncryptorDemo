FROM golang:latest AS builder
WORKDIR /go/src/github.com/xakepp35/StringEncryptorDemo
COPY go.mod .
# RUN go get -d -v go get -d net/http
COPY *.go .
RUN go build -v .
CMD ["./StringEncryptorDemo"]

# Here is demonstration of 2-stage build
#FROM alpine:latest
#RUN apk --no-cache add ca-certificates
#WORKDIR /root/
#COPY --from=builder /go/src/github.com/xakepp35/StringEncryptorDemo/StringEncryptorDemo .

#!!!!!! EXEC ERROR: alpine lacks something, need spare time to investigate.
#CMD ["./StringEncryptorDemo"]