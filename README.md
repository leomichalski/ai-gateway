# AI Gateway

REST API combining [LabStack Echo v5](https://github.com/labstack/echo) with code generation from OpenAPI 3 specifications using [`oapi-codegen`](https://github.com/oapi-codegen/oapi-codegen).

## Prerequisites

- **Go 1.25+** (required by Echo v5)

## Quick Start (For Developers)

1. **Install dependencies and generate server boilerplate**:
   ```bash
   go mod download
   make generate
   ```

1. Spin up the server.
   ```bash
   make run
   ```

1. Run tests.
   ```bash
   make test
   ```
