FROM harbor-repo.vmware.com/dockerhub-proxy-cache/progrium/busybox:latest

ADD https://curl.haxx.se/ca/cacert.pem /etc/ssl/certs/ca-certificates.crt
ADD bin /opt/resource
