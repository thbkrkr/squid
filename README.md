# Squid

Manage containers using docker-compose at scale.

![doc/img/squid-status-ui.png](doc/img/squid-status-ui.png)

## Getting started

On each node of your cluster,
start squid by given a `compose/` directory with compose files:

```
docker run -d \
  --name squid \
  --hostname=squid-$(hostname) \
  -p 4242:4242 \
  -v $(pwd)/compose:/app/compose \
  -v /var/run/docker.sock:/var/run/docker.sock \
  --restart=always \
  krkr/squid
```
