FROM scratch
LABEL name=traefik-domain
LABEL url=https://github.com/leganck/traefik-domain
COPY traefik-domain /traefik-domain
ENTRYPOINT ["/traefik-domain"]
