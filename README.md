# Magistrala Callhome Service
[![website][preview]][website]

![build][build]
![Go Report Card][grc]
[![License][LIC-BADGE]][LIC]

This is a server to receive and store information regarding Magistrala deployments. 

The summary is located on our [Website][website].

## Getting Started

### Prerequisites
- [Docker](https://docs.docker.com/get-docker/)
- [Make](https://www.gnu.org/software/make/)
- [IP to Location database](https://lite.ip2location.com/) (Required for IP geolocation features)

## Running Locally

To run the service locally with self-signed certificates:

1.  **Generate Development Certificates**:
    This command creates self-signed certificates for `localhost` and downloads necessary SSL configuration files.
    ```bash
    make dev-cert DOMAIN=localhost
    ```

    Edit the nginx.conf file and replace the domain name in the ssl_certificate and ssl_certificate_key directives with `localhost`.

2.  **Build the Docker Image**:
    ```bash
    make docker-image
    ```

3.  **Run the Service**:
    ```bash
    make run
    ```
    The service will be available at `https://localhost`.

## Deployment

For live deployment with real SSL certificates using Let's Encrypt:

1.  **Configure Domain**:
    Edit `docker/certbot/init-letsencrypt.sh` and update the `domains` variable with your actual domain name.

2.  **Generate Certificates**:
    Run the initialization script to generate Let's Encrypt certificates.
    ```bash
    cd docker
    ./certbot/init-letsencrypt.sh
    ```

3.  **Run the Service**:
    Use the Makefile to build and run the service as usual.
    ```bash
    make docker-image
    make run
    ```

### Makefile Commands
- `make docker-image`: Builds the Docker image.
- `make run`: Runs the service using Docker Compose.
- `make dev-cert`: Generates self-signed certificates for local development.
- `make clean`: Cleans up build artifacts.
- `make cleandocker`: Stops and removes Docker containers and images.
- `make test`: Runs Go tests.


## Data Collection for Magistrala
Magistrala is committed to continuously improving its services and ensuring a seamless experience for its users. To achieve this, we collect certain data from your deployments. Rest assured, this data is collected solely for the purpose of enhancing Magistrala and is not used with any malicious intent. The deployment summary can be found on our [website][website].

The collected data includes:
- **IP Address** - Used for approximate location information on deployments.
- **Services Used** - To understand which features are popular and prioritize future developments.
- **Last Seen Time** - To ensure the stability and availability of Magistrala.
- **Magistrala Version** - To track the software version and deliver relevant updates.

We take your privacy and data security seriously. All data collected is handled in accordance with our stringent privacy policies and industry best practices.

Data collection is on by default and can be disabled by setting the env variable:
`MF_SEND_TELEMETRY=false`

By utilizing Magistrala, you actively contribute to its improvement. Together, we can build a more robust and efficient IoT platform. Thank you for your trust in Magistrala!

[grc]: https://goreportcard.com/badge/github.com/absmach/callhome
[build]: https://github.com/absmach/callhome/actions/workflows/ci.yml/badge.svg
[LIC]: LICENCE
[LIC-BADGE]: https://img.shields.io/badge/License-Apache_2.0-blue.svg
[website]: https://deployments.absmach.eu
[preview]: /assets/images/website.png
