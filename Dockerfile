FROM progrium/busybox

ADD https://curl.haxx.se/ca/cacert.pem /etc/ssl/certs/ca-certificates.crt
ADD bin /opt/resource
