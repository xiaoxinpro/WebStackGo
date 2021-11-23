FROM golang:alpine AS builder
WORKDIR /build/
COPY . . 
RUN apk --update --no-cache --no-progress add git \
    && go env -w GO111MODULE=on \
    && go env -w GOPROXY=https://goproxy.cn,direct \
    && go build -o WebStackGo main.go \
    && rm public/webstack_logos.sketch \
    && mkdir /build/app/ \
    && cp -rf /build/json /build/app/ \
    && cp -rf /build/public /build/app/ \
    && cp -rf /build/views /build/app/ \
    && cp /build/WebStackGo /build/app/

FROM alpine:latest

WORKDIR /app
COPY --from=builder /build/app/ /app/

EXPOSE 2802

ENTRYPOINT ["/app/WebStackGo"]