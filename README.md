# oxidized-netbox-credential-handler
An api wrapper for handling credential sets between netbox & oxidized, written in Go!

Oxidized fundamentally cannot accept any fashion of credentials from NetBox without the username/password for the device being stored in plaintext in NetBox, and this is a small wrapper API written to rectify that issue, allowing Credential Sets to be created in NetBox's custom fields (such as `office-switch-admin`, `vps-root-creds`, etc.) that are parsed and returned to Oxidized in a fashion that it accepts. Too, with the added benefit of sterilizing NetBox's data before it reaches Oxidized. 

It's written in Go and not contributed to the Oxidized project because I can't write in Ruby and dont feel like learning it (and writing the api for fun).

Quite basic overall, but it runs as a service (can be run headless as a systemd daemon or docker service), and queries NetBox's API on behalf of Oxidized to allow `Credential Sets` defined in NetBox be parsed into valid `username:password` pairs that are returned to Oxidized for its SSH authentication. A credentials `.json` file is defined on the machine to define credential sets. 

### Security! ⚠️

It is recommended to run this as a Docker service alongside Oxidized (in a Docker container as well), so that they can share a Docker Network with only each other, this allows **only Oxidized** to query the service's API endpoint, and the socket is never even <u>bound to the host machine</u>, which is much more secure than binding it and trying to protect it.

If you do not want to run it as a Docker Container/do end up binding its listen port to the host machine, please ensure that port `:8081/tcp` or whatever else you map it to is ___properly firewalled___ !

The API token to authenticate *against* NetBox must be passed as a shell Environment variable each time you log in to restart or start the service, such that it is never hardcoded, such as:
```sh
export NETBOX_TOKEN="<key>"
```

The auth token for inbound requests to the credential wrapper's API, too, must be passed as an env var each time the service is started from a new shell

Just generate/create one as needed, whatever you pass at runtime here will be the required token coming from Oxidized for the device query requests

```sh
export WRAPPER_TOKEN="<key>"
```

## Setup & Usage Examples

Because I'm not planning on writing much config/customization into this, for any changes you'll need to adjust the raw source & recompile

You'll need go or docker on the machine to build

### Simple binary (testing)

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
WRAPPER_TOKEN="<api_token>" \
CREDENTIALS_FILE="./cred-sets.json" \

# run
./cred-wrapper
2025/06/01 01:19:52 [INFO] loaded 4 credential sets
2025/06/01 01:19:52 [INFO] cred-wrapper v0.50.3 listening on 0.0.0.0:8081
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
WRAPPER_TOKEN=xxx
CREDENTIALS_FILE=/etc/oxidized/cred-sets.json
LISTEN=127.0.0.1:8081
```

### Docker compose file (recommended)

git clone repo & copy needed files into new docker compose directory
```shell
# git clone repo
git clone https://github.com/adrian-griffin/oxidized-netbox-credential-handler.git && cd oxidized-netbox-credential-handler

# either create new docker destination dir or work out of cloned repo
# if you move to a new dir, be sure to copy needed files
mkdir /opt/docker/oxidized-cred-wrapper
```

copy needed files if you move to a new directory
```shell
# copy sourcecode files
cp main.go /opt/docker/oxidized-cred-wrapper
cp go.mod /opt/docker/oxidized-cred-wrapper

# copy docker files
cp docker-compose\ Examples/docker-compose-example.yaml /opt/docker/oxidized-cred-wrapper/docker-compose.yml
cp docker-compose\ Examples/Dockerfile-example.dockerfile /opt/docker/oxidized-cred-wrapper/Dockerfile
```

build docker image & start container
```shell
# edit your docker-compose.yml file now
> cd /opt/docker/oxidized-cred-wrapper && vim
 
# be sure to EXPORT env vars here
> docker compose up --build

Compose can now delegate builds to bake for better performance.
 To do so, set COMPOSE_BAKE=true.
[+] Building 6.7s (9/12)
 => [cred-wrapper internal] load build definition from Dockerfile
 => => transferring dockerfile: 361B
 => [cred-wrapper internal] load metadata for docker.io/library/alpine:3.20
 => [cred-wrapper internal] load metadata for docker.io/library/golang:1.22-alpine
 => [cred-wrapper internal] load .dockerignore
   .  .  .  .  
```