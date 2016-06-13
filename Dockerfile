FROM krkr/docker-toolbox

RUN apk --no-cache add bash jq && \
  curl -s https://raw.githubusercontent.com/thbkrkr/doo/1246bc77a21026e46c96dcb4cec8163f2ab7c6b6/doo \
  > /usr/local/bin/doo && chmod +x /usr/local/bin/doo

COPY views /app/views
COPY squid /app/squid

WORKDIR /app
ENTRYPOINT ["./squid"]