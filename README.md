# oxidized-netbox-credential-handler
An api wrapper for handling credential sets securely between netbox &amp; oxidized, written in Go!

## Setup & Usage Examples

### Simple binary

```shell
# build using go
git clone https://github.com/adrian-griffin/oxidized-netbox-credential-handler.git
cd oxidized-netbox-credential-handler

go build -o cred-wrapper
```

```shell
# import environment variables
NETBOX_URL="https://10.0.0.1/api/dcim/devices/?cf_oxidized_backup_bool=true&limit=0" \
NETBOX_TOKEN="<api_token>" \
CREDENTIALS_FILE="/etc/credential-sets.json" \
```

```shell
# run
> ./cred-wrapper
2025/05/31 14:00:36 [INFO] loaded 2 credential sets
2025/05/31 14:00:36 [INFO] cred-wrapper listening on 0.0.0.0:8081
```

alternatively move it to `/usr/local/bin` to allow it to be run from anywhere
```shell
sudo mv cred-wrapper /usr/local/bin/
sudo chmod 755 /usr/local/bin/cred-wrapper
```

#### Running binary as a daemon

create low-privilege user
```shell
sudo useradd -r -s /usr/sbin/nologin credwrapper
sudo chown credwrapper:credwrapper /usr/local/bin/cred-wrapper
```

create service file
```shell
# located at /etc/systemd/system/cred-wrapper.service
[Unit]
Description=Oxidized-Netbox API credential wrapper
After=network-online.target
Wants=network-online.target

[Service]
User=credwrapper
Group=credwrapper
EnvironmentFile=/etc/oxidized/cred-wrapper.env
ExecStart=/usr/local/bin/cred-wrapper
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

create env file
```shell
# located at /etc/oxidized/cred-wrapper.env
NETBOX_URL=https://netbox.abc.com/api/dcim/devices/?cf_oxidized_backup_bool=true&limit=0
NETBOX_TOKEN=xxx
CREDENTIALS_FILE=/etc/oxidized/cred-sets.json
LISTEN=127.0.0.1:8081
```

### Docker compose file example
1) export NetBox token or put it in an .env file
`export NETBOX_TOKEN="abc123"`

2) docker compose up
`docker compose up -d`        # launches both

3) check logs
`docker compose logs -f cred-wrapper`
`docker compose logs -f oxidized`

4) restart only the wrapper later
`docker compose restart cred-wrapper`
