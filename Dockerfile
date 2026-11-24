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

# Setup systemd tmpfiles to automatically create /run/notes with www-data permissions on boot
COPY notes-tmpfiles.conf /etc/tmpfiles.d/notes.conf

# Remove default nginx site config and copy custom configuration
RUN rm -f /etc/nginx/sites-enabled/default
COPY nginx.conf /etc/nginx/conf.d/notes.conf

# Setup Go application directory and copy binary and systemd service
RUN mkdir -p /opt/notes
COPY notes-app /opt/notes/notes-app
COPY notes-app.service /etc/systemd/system/notes-app.service
RUN chmod +x /opt/notes/notes-app

# Enable nginx and notes-app via systemd
RUN systemctl enable nginx notes-app

EXPOSE 80

STOPSIGNAL SIGRTMIN+3

VOLUME [ "/sys/fs/cgroup" ]

CMD ["/lib/systemd/systemd"]
