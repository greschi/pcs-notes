FROM debian:trixie-slim

ENV container=docker
ENV DEBIAN_FRONTEND=noninteractive

# Install systemd, systemd-sysv, dbus, nginx, and cleanup apt lists
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        systemd \
        systemd-sysv \
        dbus \
        nginx && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*

# Mask unnecessary systemd services that fail in containers
RUN systemctl mask \
    sys-fs-fuse-connections.mount \
    systemd-logind.service \
    getty.target \
    console-getty.service

# Create directory for the Unix socket with appropriate permissions for www-data
RUN mkdir -p /run/notes && \
    chown -R www-data:www-data /run/notes && \
    chmod 755 /run/notes

# Remove default nginx site config and copy custom configuration
RUN rm -f /etc/nginx/sites-enabled/default
COPY nginx.conf /etc/nginx/conf.d/notes.conf

# Enable nginx via systemd
RUN systemctl enable nginx

EXPOSE 80

VOLUME [ "/sys/fs/cgroup", "/run", "/run/lock" ]

CMD ["/lib/systemd/systemd"]
