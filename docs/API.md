# SDK API (v1alpha1)

## Health (Empty)

This sends a single ping to designate that the Game Server is alive and healthy. Failure to send pings within the configured thresholds will result in the GameServer being marked as `Unhealthy`.

## GetGameServer (Empty)

This returns most of the backing GameServer configuration and Status. This can be useful for instances where you may want to know Health check configuration, or the IP and Port the GameServer is currently allocated to.

And more, Game Server can get the load-balancer extern IP and Port information.

## WatchGameServer (Empty)

This will return the current GameServer details immediately whenever the underlying GameServer configuration is updated. This can be useful to track GameServer status and metadata changes, such as states, conditions, labels, annotations, and more.

## SetLabel(KeyValue)

This will set a Label value on the backing GameServer record that is stored in Kubernetes. To maintain isolation, the key value is automatically prefixed with `carrier.ocgi.dev/sdk-`.

## SetAnnotation(KeyValue)

This will set a Annotation value on the backing GameServer record that is stored in Kubernetes. To maintain isolation, the key value is automatically prefixed with `carrier.ocgi.dev/sdk-`.

## SetCondition(KeyValue)

This will set a Condition on the backing GameServer record that is stored in Kubernetes.