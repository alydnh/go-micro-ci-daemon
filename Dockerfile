FROM golang:1.14-stretch
WORKDIR /
ADD go-micro-ci-daemon .
CMD ["/go-micro-ci-daemon"]