FROM golang:1.14-stretch
WORKDIR /
ADD micro-ci .
ADD go-micro-ci-daemon .
CMD ["/go-micro-ci-daemon"]