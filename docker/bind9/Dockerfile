FROM ubuntu/bind9:latest

COPY acme-tsig.key /etc/bind/acme-tsig.key
COPY named.conf.options /etc/bind/named.conf.options
COPY named.conf.local /etc/bind/named.conf.local
COPY zones /etc/bind/zones
RUN chown -R bind:bind /etc/bind
