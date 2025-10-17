# Configuration Guide

Rhino supports flexible configuration through YAML files and environment variables.

## Configuration File

By default, Rhino looks for `config.yaml` in the current directory. You can override this with the `RHINO_CONFIG` environment variable.

### Example config.yaml

```yaml
workflows-dir: workflows
port: 8888
```

### Configuration Options

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `workflows-dir` | string | Directory containing workflow YAML files | `workflows` |
| `port` | integer | HTTP port for webhook server | `8888` |

## Environment Variables

Environment variables take precedence over values in the config file.

### Available Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `RHINO_CONFIG` | Path to custom config file | `RHINO_CONFIG=/etc/rhino/config.yaml` |
| `RHINO_WORKFLOWS_DIR` | Override workflows directory | `RHINO_WORKFLOWS_DIR=/path/to/workflows` |
| `RHINO_PORT` | Override HTTP port | `RHINO_PORT=9000` |

### Priority Order

Configuration values are loaded in the following priority order (highest to lowest):

1. Environment variables (`RHINO_PORT`, `RHINO_WORKFLOWS_DIR`)
2. Config file values (`config.yaml` or custom path)
3. Default values (if not specified)

## Usage Examples

### Using Default Configuration

```bash
./rhino runner
```

Uses `config.yaml` in the current directory.

### Using Custom Config File

```bash
RHINO_CONFIG=/etc/rhino/production.yaml ./rhino runner
```

### Using Environment Variable Overrides

```bash
# Override port only
RHINO_PORT=9000 ./rhino runner

# Override multiple values
RHINO_WORKFLOWS_DIR=/custom/workflows RHINO_PORT=9000 ./rhino runner

# Override with custom config and env vars
RHINO_CONFIG=/etc/rhino/config.yaml RHINO_PORT=9000 ./rhino runner
```

### Docker/Container Usage

```dockerfile
FROM golang:1.20

WORKDIR /app
COPY . .
RUN go build -o rhino

# Set via environment variables
ENV RHINO_WORKFLOWS_DIR=/app/workflows
ENV RHINO_PORT=8888

CMD ["./rhino", "runner"]
```

Or with docker run:

```bash
docker run -e RHINO_PORT=9000 -e RHINO_WORKFLOWS_DIR=/workflows rhino:latest
```

## Validation

Rhino validates the configuration on startup and will fail with an error if:

- Workflows directory is not set
- Port is not set or is 0
- Port is outside the valid range (1-65535)

Example validation error:

```
Error: config validation failed: port must be between 1 and 65535, got 99999
```

## Best Practices

1. **Development**: Use the default `config.yaml` for local development
2. **Production**: Use `RHINO_CONFIG` to point to a production config file
3. **Containers**: Use environment variables for container deployments
4. **CI/CD**: Use environment variables for different environments (staging, production)
5. **Security**: Never commit sensitive values to version control - use environment variables or secret management

## Troubleshooting

### Config file not found

```
Error: failed to open config file: open config.yaml: no such file or directory
```

**Solution**: Create a `config.yaml` file or set `RHINO_CONFIG` to an existing file path.

### Invalid port

```
Error: invalid port in RHINO_PORT: strconv.Atoi: parsing "abc": invalid syntax
```

**Solution**: Ensure `RHINO_PORT` is a valid integer between 1 and 65535.

### Permission denied

```
Error: failed to open config file: open config.yaml: permission denied
```

**Solution**: Ensure the config file has read permissions (at least 0644).