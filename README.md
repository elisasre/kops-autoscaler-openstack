# kops-autoscaler-openstack

The purpose of this application is to provide capability to scale cluster up/down in case of need. This application supports currently two use-cases:

- if kops instancegroup minsize is larger than current instances in openstack -> scale up 
- if kops instancegroup maxsize is smaller than current instances in openstack -> scale down

This application will detect the need of change by running `kops update cluster <cluster>`. Scaling means that this application will execute `kops update cluster <cluster> --yes` under the hood.

This application makes it possible to use `kops rolling-update <cluster>` command in openstack kops. 

```
Provide autoscaling capability to kops openstack

Usage:
  kops-autoscaling-openstack [flags]

Flags:
      --access-key string        S3 access key (default "test")
      --custom-endpoint string   S3 custom endpoint
  -h, --help                     help for kops-autoscaling-openstack
      --name string              Name of the kubernetes kops cluster (default "")
      --secret-key string        S3 secret key (default "")
      --sleep int                Sleep between executions (default 45)
      --state-store string       KOPS State store (default "")
```

### Copying bindata from kops

Kops needs its bindata to be compiled and copied under vendor folder.

```
make copybindata
make ensure
```

### How to install

See Examples

Create secrets, set env variables that you usually use when doing things with kops. 

### How to contribute

Make issues/PRs

### Debug in Production

This binary do have pprof debugging options available in localhost:6060/debug/pprof/ by default.
When this binary is executed inside kubernetes cluster, port-forward that port to your localhost and start debugging.
Command for that is `kubectl port-forward <pod> 6060:6060`
