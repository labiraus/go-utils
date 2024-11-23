# basicapi

This is a simple golang api designed to be run in a kuberenetes cluster.

## Dependencies

[go-common/api](/apps/go-common/api) provides `/readiness` and `/liveness` endpoints, as well as graceful shutdown

## Endpoints

### /hello

Dummy user lookup endpoint - always returns username based on kubernetes secret if available and email "something@somewhere.com"

#### Request

> HTTP POST

``` json
{
    "userid": "int"
}
```

#### Response

``` json
{
    "userid": "int",
    "username": "string",
    "email": "string,"
}
```

#### Curl

``` bash
curl -X POST http://localhost:8080/hello -d '{"userid": 1}'
```
