# oxidized-netbox-credential-handler
An api wrapper for handling credential sets securely between netbox &amp; oxidized, written in Go!

## Setup & Usage Examples

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