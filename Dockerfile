FROM scratch

ADD ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
ADD shadow /etc/shadow
ADD bin /opt/resource
