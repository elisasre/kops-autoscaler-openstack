module github.com/elisasre/kops-autoscaler-openstack

go 1.20

replace (
	k8s.io/api => k8s.io/api v0.28.1
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.28.1
	k8s.io/apimachinery => k8s.io/apimachinery v0.28.1
	k8s.io/apiserver => k8s.io/apiserver v0.28.1
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.28.1
	k8s.io/client-go => k8s.io/client-go v0.28.1
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.28.1
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.28.1
	k8s.io/code-generator => k8s.io/code-generator v0.28.1
	k8s.io/component-base => k8s.io/component-base v0.28.1
	k8s.io/component-helpers => k8s.io/component-helpers v0.28.1
	k8s.io/controller-manager => k8s.io/controller-manager v0.28.1
	k8s.io/cri-api => k8s.io/cri-api v0.28.1
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.28.1
	k8s.io/kops => k8s.io/kops v1.28.0-beta.1.0.20230904042947-8bf26125c271
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.28.1
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.28.1
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.28.1
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.28.1
	k8s.io/kubectl => k8s.io/kubectl v0.28.1
	k8s.io/kubelet => k8s.io/kubelet v0.28.1
	k8s.io/kubernetes => k8s.io/kubernetes v1.28.1
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.28.1
	k8s.io/metrics => k8s.io/metrics v0.28.1
	k8s.io/mount-utils => k8s.io/mount-utils v0.28.1
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.28.1
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.28.1
)

require (
	github.com/golang/glog v1.1.1
	github.com/gophercloud/gophercloud v1.6.0
	github.com/prometheus/client_golang v1.16.0
	github.com/spf13/cobra v1.7.0
	k8s.io/apimachinery v0.28.1
	k8s.io/kops v0.0.0-00010101000000-000000000000
)

require (
	cloud.google.com/go/compute v1.23.0 // indirect
	cloud.google.com/go/compute/metadata v0.2.3 // indirect
	github.com/Azure/azure-pipeline-go v0.2.3 // indirect
	github.com/Azure/azure-sdk-for-go v68.0.0+incompatible // indirect
	github.com/Azure/azure-storage-blob-go v0.15.0 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.29 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.23 // indirect
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.12 // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.6 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/autorest/to v0.4.0 // indirect
	github.com/Azure/go-autorest/autorest/validation v0.3.1 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/GoogleCloudPlatform/k8s-cloud-provider v1.24.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver/v3 v3.2.1 // indirect
	github.com/Masterminds/sprig/v3 v3.2.3 // indirect
	github.com/apparentlymart/go-cidr v1.1.0 // indirect
	github.com/aws/aws-sdk-go v1.45.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.14.3 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/digitalocean/godo v1.102.1 // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/docker/cli v24.0.5+incompatible // indirect
	github.com/docker/distribution v2.8.2+incompatible // indirect
	github.com/docker/docker v24.0.5+incompatible // indirect
	github.com/docker/docker-credential-helpers v0.8.0 // indirect
	github.com/emicklei/go-restful/v3 v3.10.2 // indirect
	github.com/evanphx/json-patch v5.6.0+incompatible // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-logr/logr v1.2.4 // indirect
	github.com/go-openapi/jsonpointer v0.20.0 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/swag v0.22.4 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/google/gnostic-models v0.6.8 // indirect
	github.com/google/go-cmp v0.5.9 // indirect
	github.com/google/go-containerregistry v0.16.1 // indirect
	github.com/google/go-querystring v1.1.0 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/s2a-go v0.1.5 // indirect
	github.com/google/uuid v1.3.1 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.2.5 // indirect
	github.com/googleapis/gax-go/v2 v2.12.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.2 // indirect
	github.com/hashicorp/go-retryablehttp v0.7.4 // indirect
	github.com/hetznercloud/hcloud-go v1.50.0 // indirect
	github.com/huandu/xstrings v1.4.0 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-ieproxy v0.0.11 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/sys/mountinfo v0.6.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0-rc4 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pkg/sftp v1.13.6 // indirect
	github.com/prometheus/client_model v0.4.0 // indirect
	github.com/prometheus/common v0.44.0 // indirect
	github.com/prometheus/procfs v0.11.1 // indirect
	github.com/scaleway/scaleway-sdk-go v1.0.0-beta.20.0.20230829075301-d16b548612e4 // indirect
	github.com/sergi/go-diff v1.3.1 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/spf13/cast v1.5.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spotinst/spotinst-sdk-go v1.145.0 // indirect
	github.com/vbatts/tar-split v0.11.5 // indirect
	go.opencensus.io v0.24.0 // indirect
	golang.org/x/crypto v0.12.0 // indirect
	golang.org/x/exp v0.0.0-20230817173708-d852ddb80c63 // indirect
	golang.org/x/net v0.14.0 // indirect
	golang.org/x/oauth2 v0.11.0 // indirect
	golang.org/x/sync v0.3.0 // indirect
	golang.org/x/sys v0.11.0 // indirect
	golang.org/x/term v0.11.0 // indirect
	golang.org/x/text v0.12.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	google.golang.org/api v0.138.0 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230807174057-1744710a1577 // indirect
	google.golang.org/grpc v1.57.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
	gopkg.in/gcfg.v1 v1.2.3 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/square/go-jose.v2 v2.6.0 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/api v0.28.1 // indirect
	k8s.io/client-go v0.28.1 // indirect
	k8s.io/cloud-provider v0.28.1 // indirect
	k8s.io/cloud-provider-aws v1.28.1 // indirect
	k8s.io/cloud-provider-gcp/providers v0.27.1 // indirect
	k8s.io/component-base v0.28.1 // indirect
	k8s.io/component-helpers v0.28.1 // indirect
	k8s.io/csi-translation-lib v0.28.1 // indirect
	k8s.io/klog/v2 v2.100.1 // indirect
	k8s.io/kube-openapi v0.0.0-20230717233707-2695361300d9 // indirect
	k8s.io/mount-utils v0.28.1 // indirect
	k8s.io/utils v0.0.0-20230726121419-3b25d923346b // indirect
	sigs.k8s.io/json v0.0.0-20221116044647-bc3834ca7abd // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.3.0 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)
