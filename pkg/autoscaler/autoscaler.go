package autoscaler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	// import pprof package, needed for debugging
	_ "net/http/pprof"

	"github.com/golang/glog"
	"github.com/gophercloud/gophercloud"
	openstackv2 "github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/startstop"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/loadbalancers"
	"github.com/gophercloud/gophercloud/openstack/loadbalancer/v2/pools"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kops/pkg/apis/kops"
	"k8s.io/kops/pkg/client/simple"
	"k8s.io/kops/pkg/client/simple/vfsclientset"
	"k8s.io/kops/upup/pkg/fi"
	"k8s.io/kops/upup/pkg/fi/cloudup"
	"k8s.io/kops/upup/pkg/fi/cloudup/openstack"
	"k8s.io/kops/util/pkg/vfs"
)

// Options contains startup variables from cobra cmd
type Options struct {
	LogLevel            string
	Sleep               int
	LoadBalancerMetrics bool
	StateStore          string
	AccessKey           string
	SecretKey           string
	CustomEndpoint      string
	ClusterName         string
}

type openstackASG struct {
	ApplyCmd  *cloudup.ApplyClusterCmd
	clientset simple.Clientset
	opts      *Options
	Cloud     fi.Cloud
}

var (
	osInstances = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "openstack_instance",
			Help: "Openstack instance",
		},
		[]string{"name", "id", "status"},
	)
	lbActiveConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "load_balancer_active_connections",
			Help: "Load balancer active connections",
		},
		[]string{"name", "id"},
	)
	lbBytesIn = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "load_balancer_bytes_in",
			Help: "Load balancer bytes in",
		},
		[]string{"name", "id"},
	)
	lbBytesOut = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "load_balancer_bytes_out",
			Help: "Load balancer bytes out",
		},
		[]string{"name", "id"},
	)
	lbRequestErros = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "load_balancer_request_errors",
			Help: "Load balancer request errors",
		},
		[]string{"name", "id"},
	)
	lbTotalConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "load_balancer_total_connections",
			Help: "Load balancer total connections",
		},
		[]string{"name", "id"},
	)
	lbStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "openstack_loadbalancer",
			Help: "Shows the OpenStack loadbalancers",
		},
		[]string{"name", "id", "provisioning_status", "operating_status"},
	)
	lbMemberWeight = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "load_balancer_member_weight",
			Help: "Load balancer member weight",
		},
		[]string{"name", "id", "pool_name", "pool_id", "provisioning_status", "operating_status"},
	)
)

// Run will execute cluster check in loop periodically
func Run(opts *Options) error {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	registryBase, err := vfs.Context.BuildVfsPath(opts.StateStore)
	if err != nil {
		return fmt.Errorf("error parsing registry path %q: %v", opts.StateStore, err)
	}

	clientset := vfsclientset.NewVFSClientset(registryBase)
	osASG := &openstackASG{
		opts:      opts,
		clientset: clientset,
	}

	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":2112", nil)

	prometheus.MustRegister(lbActiveConnections)
	prometheus.MustRegister(lbBytesIn)
	prometheus.MustRegister(lbBytesOut)
	prometheus.MustRegister(lbRequestErros)
	prometheus.MustRegister(lbTotalConnections)
	prometheus.MustRegister(lbStatus)
	prometheus.MustRegister(lbMemberWeight)

	fails := 0
	for {
		if fails > 5 {
			return fmt.Errorf("Too many failed attempts")
		}
		ctx := context.Background()
		time.Sleep(time.Duration(opts.Sleep) * time.Second)
		glog.V(2).Infof("Executing...\n")

		if osASG.Cloud == nil {
			cluster, err := osASG.clientset.GetCluster(ctx, osASG.opts.ClusterName)
			if err != nil {
				return fmt.Errorf("error initializing cluster %v", err)
			}

			cloud, err := cloudup.BuildCloud(cluster)
			if err != nil {
				return err
			}
			osASG.Cloud = cloud
		}

		err := osASG.updateApplyCmd(ctx)
		if err != nil {
			glog.Errorf("Error updating applycmd %v", err)
			fails++
			continue
		}

		needsUpdate, err := osASG.dryRun()
		if err != nil {
			glog.Errorf("Error running dryrun %v", err)
			fails++
			continue
		}

		if needsUpdate {
			// ApplyClusterCmd is always appending assets
			// so when dryrun is executed first the assets will be duplicated if we do not set it nil here
			osASG.ApplyCmd.Assets = nil
			err = osASG.update(ctx)
			if err != nil {
				glog.Errorf("Error updating cluster %v", err)
				fails++
				continue
			}
		}

		// Collecting load balancer metrics is not critical. Don't want to fail on error.
		if opts.LoadBalancerMetrics {
			osASG.enableMetrics()
		}

		fails = 0
	}
}

func (osASG *openstackASG) updateApplyCmd(ctx context.Context) error {
	cluster, err := osASG.clientset.GetCluster(ctx, osASG.opts.ClusterName)
	if err != nil {
		return fmt.Errorf("error initializing cluster %v", err)
	}

	list, err := osASG.clientset.InstanceGroupsFor(cluster).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}
	var instanceGroups []*kops.InstanceGroup
	for i := range list.Items {
		instanceGroups = append(instanceGroups, &list.Items[i])
	}

	osASG.ApplyCmd = &cloudup.ApplyClusterCmd{
		Cloud:          osASG.Cloud,
		Clientset:      osASG.clientset,
		Cluster:        cluster,
		InstanceGroups: instanceGroups,
		Phase:          "",
		TargetName:     cloudup.TargetDryRun,
		OutDir:         "out",
	}
	return nil
}

// dryRun scans do we need run update or not
// currently it supports scaling up and down instances
// we do not use kops update cluster dryrun because it will make lots of API queries against OpenStack.
func (osASG *openstackASG) dryRun() (bool, error) {
	osCloud := osASG.Cloud.(openstack.OpenstackCloud)
	instances, err := osCloud.ListInstances(servers.ListOpts{})
	if err != nil {
		return false, err
	}
	currentIGs := make(map[string]int32)
	cluster := osASG.ApplyCmd.Cluster
	instanceGroups := osASG.ApplyCmd.InstanceGroups

	for _, ig := range instanceGroups {
		currentIGs[ig.Name] = 0
	}

	osInstances.Reset()
	for _, instance := range instances {
		val, ok := instance.Metadata["k8s"]
		ig, ok2 := instance.Metadata["KopsInstanceGroup"]
		if ok && ok2 && val == cluster.Name {
			maintenanceVal, ok3 := instance.Metadata["maintenance"]
			if instance.Status == "SHUTOFF" && (!ok3 || maintenanceVal != "true") {
				startErr := startstop.Start(osCloud.ComputeClient(), instance.ID).ExtractErr()
				if startErr != nil {
					glog.Errorf("Could not start server %v", startErr)
				} else {
					glog.V(2).Infof("Starting server %s (%s)", instance.Name, instance.ID)
				}
			}
			osInstances.WithLabelValues(instance.Name, instance.ID, instance.Status).Set(1)
			currentVal, found := currentIGs[ig]
			if found {
				currentIGs[ig] = currentVal + 1
			} else {
				glog.Errorf("Error found instancegroup %s which does not exist anymore", ig)
			}
		}
	}

	for _, ig := range instanceGroups {
		if fi.Int32Value(ig.Spec.MinSize) < currentIGs[ig.Name] {
			glog.V(2).Infof("Scaling down running update --yes")
			return true, nil
		}
		if fi.Int32Value(ig.Spec.MinSize) > currentIGs[ig.Name] {
			glog.V(2).Infof("Scaling up running update --yes")
			return true, nil
		}
	}
	return false, nil
}

func (osASG *openstackASG) update(ctx context.Context) error {
	osASG.ApplyCmd.TargetName = cloudup.TargetDirect
	osASG.ApplyCmd.DryRun = false
	var options fi.RunTasksOptions
	options.InitDefaults()
	osASG.ApplyCmd.RunTasksOptions = &options
	if err := osASG.ApplyCmd.Run(ctx); err != nil {
		return err
	}
	return nil
}

func (osASG *openstackASG) enableMetrics() error {
	authOpts, err := openstackv2.AuthOptionsFromEnv()
	if err != nil {
		glog.Errorf("Error building auth options from env %v", err)
		return err
	}
	provider, err := openstackv2.AuthenticatedClient(authOpts)
	if err != nil {
		glog.Errorf("Error building openstack authenticated client %v", err)
		return err
	}
	client, err := openstackv2.NewLoadBalancerV2(provider, gophercloud.EndpointOpts{})
	if err != nil {
		glog.Errorf("Error building openstack load balancer client %v", err)
		return err
	}

	err = osASG.getLoadBalancerMetrics(client)
	if err != nil {
		return err
	}

	err = osASG.getMemberMetrics(client)
	if err != nil {
		return err
	}

	return nil
}

func (osASG *openstackASG) getLoadBalancerMetrics(client *gophercloud.ServiceClient) error {
	allPages, err := loadbalancers.List(client, loadbalancers.ListOpts{}).AllPages()
	if err != nil {
		glog.Errorf("Error listing load balancer pages %v", err)
		return err
	}
	allLoadBalancers, err := loadbalancers.ExtractLoadBalancers(allPages)
	if err != nil {
		glog.Errorf("Error extracting load balancers %v", err)
		return err
	}

	lbStatus.Reset()
	for _, lb := range allLoadBalancers {
		lbStatus.WithLabelValues(lb.Name, lb.ID, lb.ProvisioningStatus, lb.OperatingStatus).Set(float64(1))
		stats, err := loadbalancers.GetStats(client, lb.ID).Extract()
		if err != nil {
			glog.Errorf("Error getting load balancer stats %v", err)
			continue
		}
		glog.V(4).Infof("Load balancer statistics collected %s", lb.Name)

		lbActiveConnections.WithLabelValues(lb.Name, lb.ID).Set(float64(stats.ActiveConnections))
		lbBytesIn.WithLabelValues(lb.Name, lb.ID).Set(float64(stats.BytesIn))
		lbBytesOut.WithLabelValues(lb.Name, lb.ID).Set(float64(stats.BytesOut))
		lbRequestErros.WithLabelValues(lb.Name, lb.ID).Set(float64(stats.RequestErrors))
		lbTotalConnections.WithLabelValues(lb.Name, lb.ID).Set(float64(stats.TotalConnections))
	}

	return nil
}

func (osASG *openstackASG) getMemberMetrics(client *gophercloud.ServiceClient) error {
	allPages, err := pools.List(client, pools.ListOpts{}).AllPages()
	if err != nil {
		glog.Errorf("Error listing load balancer pool pages %v", err)
		return err
	}
	allPools, err := pools.ExtractPools(allPages)
	if err != nil {
		glog.Errorf("Error extracting load balancer pools %v", err)
		return err
	}

	for _, pool := range allPools {
		allPages, err := pools.ListMembers(client, pool.ID, pools.ListMembersOpts{}).AllPages()
		if err != nil {
			glog.Errorf("Error listing load balancer member pages %v", err)
			return err
		}
		allMembers, err := pools.ExtractMembers(allPages)
		if err != nil {
			glog.Errorf("Error extracting load balancer members %v", err)
			return err
		}
		glog.V(4).Infof("Load balancer member status collected for pool %s", pool.Name)

		for _, member := range allMembers {
			lbMemberWeight.WithLabelValues(member.Name, member.ID, pool.Name, pool.ID, member.ProvisioningStatus, member.OperatingStatus).Set(float64(member.Weight))
		}
	}

	return nil
}
