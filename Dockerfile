FROM alpine:latest

MAINTAINER Edward Muller <edward@heroku.com>

WORKDIR "/opt"

ADD .docker_build/Assignment1 /opt/bin/Assignment2

CMD ["/opt/bin/Assignment2"]
