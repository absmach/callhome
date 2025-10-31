# Magistrala Callhome Service
[![website][preview]][website]

![build][build]
![Go Report Card][grc]
[![License][LIC-BADGE]][LIC]

This is a server to receive and store information regarding Magistrala deployments. 

The summary is located on our [Website][website].

## Usage
To Run:

```bash
make docker-image
make run
```


### Requirements
- [IP to Location database](https://lite.ip2location.com/)

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
