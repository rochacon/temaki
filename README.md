# temaki

Disposable test environment for 12 factor applications.

`temaki` depends on Docker.

:warning: `temaki` is still in early development and may/will change.

## Install

Just `go get` it:

```
go get github.com/rochacon/temaki
```

## Using it

### Create a `temaki.yml` file at your project root folder

Here is a example of the config file

```yaml
-- The command that run your test suite
cmd: /usr/bin/env
-- Dockerfile location
dockerfile: ./Dockerfile
-- The command that run your test suite
image: my-app
env:
  -- those are the environment variables that Temaki will set for you test suite
  DATABASE_URL:
    format: postgres://postgres:postgres@{{ .Host }}:{{ .Port }}/test?sslmode=disable
    image: postgres:9.4
    port: 5432
    hooks:
      pre-run:
        - psql "CREATE DATABASE test CHARSET utf8;"
      post-run:
        - psql "DROP DATABASE test;"
  BROKER_URL:
      format: amqp://guest@{{ .Host }}:{{ .Port }}/test
      image: rabbitmq
```

Now run `temaki` to run your test suite against disposable services.


You may also specify the test suite command at runtime, like:

```
temaki go test -cover -v ./handlers/...
```

## TODO

- [X] Use `DOCKER_HOST` environment variable
- [X] Implement test hooks using `docker exec`
- [ ] Support multiple port bindings
- [ ] Tests!
