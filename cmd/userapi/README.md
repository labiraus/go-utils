# userapi

This is a simple golang api designed to be run in a kuberenetes cluster.

## Dependencies

[go-common/api](/apps/go-common/api) provides `/readiness` and `/liveness` endpoints, as well as graceful shutdown

## Endpoints

### /user

Dummy user lookup endpoint - always returns username "a nonny mouse" and email "something@somewhere.com"

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