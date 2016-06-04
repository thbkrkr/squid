FROM krkr/dops

COPY views /app/views
COPY squid /app/squid

ENTRYPOINT ["/app/squid"]