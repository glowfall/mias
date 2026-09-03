# Production deployment

The production setup deliberately gives each TCP port a single owner:

- nginx listens publicly on ports 80 and 443;
- `asot` listens only on `127.0.0.1:8080`;
- Certbot uses the nginx-served webroot for ACME challenges;
- certificate renewal reloads nginx and never stops `asot`.

This avoids the outage seen on 2026-06-23, when a package update started nginx while the standalone Certbot hook had stopped `asot`. Nginx claimed port 80, renewal failed, and `asot` could no longer restart.

## Deploy the application

Run these commands from the repository root on the server:

```sh
go test ./...
go build -o /root/go/bin/asot .

install -m 0644 deploy/systemd/asot.service /etc/systemd/system/asot.service
systemctl daemon-reload
systemctl enable --now asot.service
```

The health check below must return an HTTP success response before nginx is reloaded:

```sh
curl --fail --silent --show-error \
  --header 'Host: www.mias.top' \
  http://127.0.0.1:8080/ \
  --output /dev/null
```

## Configure nginx

The TLS configuration expects an existing certificate at `/etc/letsencrypt/live/mias.top`. On a new host, bootstrap that certificate before installing the TLS configuration.

```sh
install -d -m 0755 /var/www/letsencrypt/.well-known/acme-challenge
install -m 0644 deploy/nginx/mias.top.conf /etc/nginx/sites-available/mias.top
ln -sfn /etc/nginx/sites-available/mias.top /etc/nginx/sites-enabled/mias.top

nginx -t
systemctl enable nginx.service
systemctl reload nginx.service
```

## Configure certificate renewal

Use the snap version of Certbot as the single renewal mechanism. The first command is a one-time migration of the existing lineage from the standalone authenticator to the webroot authenticator. Do not routinely force renewals; afterward, let the timer renew only when the certificate is due.

```sh
install -m 0755 \
  deploy/letsencrypt/renewal-hooks/deploy/reload-nginx \
  /etc/letsencrypt/renewal-hooks/deploy/reload-nginx

/snap/bin/certbot certonly \
  --webroot \
  --webroot-path /var/www/letsencrypt \
  --cert-name mias.top \
  -d mias.top \
  -d www.mias.top \
  --force-renewal \
  --non-interactive

systemctl disable --now certbot.timer
systemctl mask certbot.timer
systemctl enable --now snap.certbot.renew.timer

/snap/bin/certbot renew \
  --dry-run \
  --cert-name mias.top \
  --no-random-sleep-on-renew
```

Do not add Certbot pre/post hooks that stop and start `asot`. ACME challenges are served by nginx and do not require downtime.

## Verify production

```sh
systemctl is-active asot nginx snap.certbot.renew.timer
ss -lntp | grep -E ':(80|443|8080)[[:space:]]'
curl --fail --silent --show-error https://www.mias.top/ --output /dev/null
```

Expected listeners are nginx on public ports 80 and 443, and `asot` on `127.0.0.1:8080` only.
