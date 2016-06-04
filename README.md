# Squid

An API to deploy containers using docker-compose.

## Getting started

Run squid on a node.

```
docker run -d \
  -p 8442:4242 \
  -v $(pwd)/composes:/app/composes \
  -v /var/run/docker.sock:/var/run/docker.sock \
  krkr/squid
```