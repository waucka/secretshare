FROM alpine:latest

MAINTAINER Alex Wauck "alexwauck@exosite.com"
EXPOSE 5000

ENV GIN_MODE release

COPY build/linux-amd64/secretshare-server /usr/bin/secretshare-server
COPY docker-secretshare-server.json /etc/secretshare-server.json
RUN mkdir /root/.aws
RUN chmod 0700 /root/.aws
COPY aws_creds /root/.aws/credentials
RUN chmod 0600 /root/.aws/credentials

CMD ["/usr/bin/secretshare-server"]
