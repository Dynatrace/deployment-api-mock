# Deployment API Mock

Web application that provides an endpoint to query for OneAgent packages for testing purposes.

## Requirements

- Go 1.11

## Usage

The project can be built with `go build` and then running the created binary. The web server will run at port 8080.

To use the API, you first need to register a Query:

`POST /register`

There are several options you can set on the payload for this request:

| Key           | Description                                                                | Required |
| ------------- | -------------------------------------------------------------------------- | -------- |
| platform      | Platform: `unix`, `windows`                                                | Yes      |
| installerType | Package type: `default`, `default-unattended`                              | Yes      |
| apiToken      | Token to register the query with.                                          | Yes      |
| waitTime      | How long to wait before returning a response, e.g. `30m`. Default: no wait | No       |
| exitCode      | Exit code to be returned by the installer. Default: 0                      | No       |

Currently the following platform/installerType pairs are supported, any others would fail:
- `unix` / `default`
- `windows` / `default-unattended`

Once a Query has been registered, any subsequent GET requests for the below pattern would return the configured installer package.

`GET /v1/deployment/installer/agent/{platform}/{installerType}/latest?Api-Token={apiToken}`

Any requests for Queries that haven't been registered previously will fail with 404 HTTP Code.
