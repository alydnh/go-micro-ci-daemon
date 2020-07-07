FROM golang:1.14-stretch
WORKDIR /
COPY micro-ci /micro-ci
ADD go-micro-ci-daemon .
CMD ["/go-micro-ci-daemon"]