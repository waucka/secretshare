FROM debian:latest

MAINTAINER Alexander Wauck "alex@impulse101.org"
EXPOSE 5000

ENV GIN_MODE release

RUN apt-get update && apt-get upgrade -y && apt-get -y install curl python2.7

RUN curl -L -o /usr/bin/secretshare-server 'https://github.com/waucka/secretshare/releases/download/1.0.0/linux-secretshare-server'
RUN chmod 0755 /usr/bin/secretshare-server

COPY run-secretshare-server.py /usr/bin/run-secretshare-server
RUN chmod 0755 /usr/bin/run-secretshare-server

RUN ln -s /usr/bin/python2.7 /usr/bin/python2

CMD ["/usr/bin/run-secretshare-server"]
