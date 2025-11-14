# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [0.53.1] - 2025-11-13
- Custom HTTP handler to allow internal CA/self-signed SSL certificates on HTTP endpoints
- Logging tweaks to HTTP handler and JSON unmarshalling
- Added support for setting fixed name resolution to docker-compose

## [0.52.1] - 2025-6-16
- Added dynamic SSH port mapping from NetBox

## [0.51.0] - 2025-6-6
- Added enable password support (`cf.enable_password` in netbox)
- Added inbound API auth/token validation for requests

## [0.50.4] - 2025-6-1
- /healthz tweaks 
- README updates

## [0.50.3] - 2025-5-31
- Adding healthz endpoint

## [0.50.2] - 2025-5-31
- Request src IP logging
- Adjustments to Dockerfile & compose example config

## [0.50.1] - 2025-5-31
- Begin changelog & versioning
