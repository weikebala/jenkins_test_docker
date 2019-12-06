FROM golang:latest

ADD .  /goapp/


WORKDIR /goapp
ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.io
RUN go build
EXPOSE 9001
CMD ./short_link

