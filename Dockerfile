FROM golang:1.15-alpine as builder

WORKDIR /master

COPY .  .

RUN GO111MODULE=on GOPROXY=https://goproxy.cn,direct CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o go_crontab_master master/main/master.go && \
    mkdir app && cp go_crontab_master app && cp master/main/master.json app && \
    cp -r master/main/webroot app && go build -o go_crontab_worker worker/main/worker.go && \
    cp go_crontab_worker app && cp worker/main/worker.json app

FROM alpine

WORKDIR	/master

RUN echo "https://mirror.tuna.tsinghua.edu.cn/alpine/v3.4/main/" > /etc/apk/repositories
RUN apk update && apk upgrade && apk add bash

COPY --from=builder /master/app .

EXPOSE	8070

CMD ./go_crontab_master
