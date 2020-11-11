## HTTP API

* GetGameServer

```
GET /gameserver
```

Example:

```shell
# curl -X GET "http://localhost:9021/gameserver" -H "accept: application/json"
```

* WatchGameServer

```
GET /watch/gameserver
```

Example:

```shell
# curl -X GET "http://localhost:9021/watch/gameserver" -H "accept: application/json"
```

* SetLabel

```
PUT /metadata/label
```

Example:

```shell
# curl -X PUT "http://localhost:9021/metadata/label" -H "accept: application/json" -H "Content-Type: application/json" -d '{"key": "foo", "value": "bar"}'
```

* SetAnnotation

```
PUT /metadata/annotation
```

Example:

```shell
# curl -X PUT "http://localhost:9021/metadata/annotation" -H "accept: application/json" -H "Content-Type: application/json" -d '{"key": "foo", "value": "bar"}'
```

* SetCondition

```
PUT /condition

```

Example:

```shell
# curl -X PUT "http://localhost:9021/condition" -H "accept: application/json" -H "Content-Type: application/json" -d '{"key": "carrier.ocgi.dev/server-ready", "value": "True"}'
```

* Health

```
POST /health
```

Example:

```shell
# curl -X POST "http://localhost:9021/health"
```