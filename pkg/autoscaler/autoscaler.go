package autoscaler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	// Import pprof package, needed for debugging.
	_ "net/http/pprof" //nolint: gosec

	"github.com/golang/glog"
	"github.com/gophercloud/gophercloud/v2"
	openstackv2 "github.com/gophercloud/gophercloud/v2/openstack"
	cinderquota "github.com/gophercloud/gophercloud/v2/openstack/blockstorage/v3/quotasets"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/quotasets"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/loadbalancers"
	"github.com/gophercloud/gophercloud/v2/openstack/loadbalancer/v2/pools"
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

// Options contains startup variables from cobra cmd.
type Options struct {
	LogLevel            string
	Sleep               int
	LoadBalancerMetrics bool
	QuotaMetrics        bool
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
	lbPoolMember = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "load_balancer_pool_member",
			Help: "Load balancer pool member",
		},
		[]string{
			"name", "id", "pool_name", "pool_id", "load_balancer_id",
			"provisioning_status", "operating_status", "weight",
		},
	)

	ramUsed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "openstack_ram_used",
			Help: "Openstack ram used",
		},
		[]string{"project_id", "project_name"},
	)

	ramQuota = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "openstack_ram_quota",
			Help: "Openstack ram quota",
		},
		[]string{"project_id", "project_name"},
	)

	secGroupsUsed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "openstack_security_groups_used",
			Help: "Openstack security groups used",
		},
		[]string{"project_id", "project_name"},
	)

	secGroupsQuota = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "openstack_security_groups_quota",
			Help: "Openstack security groups quota",
		},
		[]string{"project_id", "project_name"},
	)

	coreUsed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "openstack_cores_used",
			Help: "Openstack cores used",
		},
		[]string{"project_id", "project_name"},
	)

	coreQuota = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "openstack_cores_quota",
			Help: "Openstack cores quota",
		},
		[]string{"project_id", "project_name"},
	)

	instancesUsed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "openstack_instances_used",
			Help: "Openstack instances used",
		},
		[]string{"project_id", "project_name"},
	)

	instancesQuota = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "openstack_instances_quota",
			Help: "Openstack instances quota",
		},
		[]string{"project_id", "project_name"},
	)

	serverGroupsUsed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "openstack_server_groups_used",
			Help: "Openstack server groups used",
		},
		[]string{"project_id", "project_name"},
	)

	serverGroupsQuota = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "openstack_server_groups_quota",
			Help: "Openstack server groups quota",
		},
		[]string{"project_id", "project_name"},
	)

	volumesUsed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "openstack_volumes_used",
			Help: "Openstack volumes used",
		},
		[]string{"project_id", "project_name"},
	)

	volumesQuota = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "openstack_volumes_quota",
			Help: "Openstack volumes quota",
		},
		[]string{"project_id", "project_name"},
	)

	spaceUsed = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "openstack_volume_gigabytes_used",
			Help: "Openstack space used",
		},
		[]string{"project_id", "project_name"},
	)

	spaceQuota = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "openstack_volume_gigabytes_quota",
			Help: "Openstack space quota",
		},
		[]string{"project_id", "project_name"},
	)
)

// Run will execute cluster check in loop periodically.
func Run(opts *Options) error {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil)) //nolint: gosec
	}()

	registryBase, err := vfs.Context.BuildVfsPath(opts.StateStore)
	if err != nil {
		return fmt.Errorf("error parsing registry path %q: %w", opts.StateStore, err)
	}

	clientset := vfsclientset.NewVFSClientset(vfs.Context, registryBase)
	osASG := &openstackASG{
		opts:      opts,
		clientset: clientset,
	}

	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":2112", nil) //nolint: errcheck,gosec

	prometheus.MustRegister(lbActiveConnections)
	prometheus.MustRegister(lbBytesIn)
	prometheus.MustRegister(lbBytesOut)
	prometheus.MustRegister(lbRequestErros)
	prometheus.MustRegister(lbTotalConnections)
	prometheus.MustRegister(lbStatus)
	prometheus.MustRegister(lbPoolMember)
	prometheus.MustRegister(ramUsed)
	prometheus.MustRegister(ramQuota)
	prometheus.MustRegister(secGroupsUsed)
	prometheus.MustRegister(secGroupsQuota)
	prometheus.MustRegister(coreUsed)
	prometheus.MustRegister(coreQuota)
	prometheus.MustRegister(instancesUsed)
	prometheus.MustRegister(instancesQuota)
	prometheus.MustRegister(serverGroupsUsed)
	prometheus.MustRegister(serverGroupsQuota)
	prometheus.MustRegister(volumesUsed)
	prometheus.MustRegister(volumesQuota)
	prometheus.MustRegister(spaceUsed)
	prometheus.MustRegister(spaceQuota)

	fails := 0
	for {
		if fails > 5 {
			return fmt.Errorf("too many failed attempts")
		}
		ctx := context.Background()
		time.Sleep(time.Duration(opts.Sleep) * time.Second)
		glog.V(2).Infof("Executing...\n")

		if osASG.Cloud == nil {
			cluster, err := osASG.clientset.GetCluster(ctx, osASG.opts.ClusterName)
			if err != nil {
				return fmt.Errorf("error initializing cluster %w", err)
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
			err = osASG.update(ctx)
			if err != nil {
				glog.Errorf("Error updating cluster %v", err)
				fails++
				continue
			}
		}

		// Collecting load balancer / quota metrics is not critical. Don't want to fail on error.
		if opts.LoadBalancerMetrics || opts.QuotaMetrics {
			_ = osASG.enableMetrics(opts.LoadBalancerMetrics, opts.QuotaMetrics)
		}

		fails = 0
	}
}

func (osASG *openstackASG) updateApplyCmd(ctx context.Context) error {
	cluster, err := osASG.clientset.GetCluster(ctx, osASG.opts.ClusterName)
	if err != nil {
		return fmt.Errorf("error initializing cluster %w", err)
	}

	list, err := osASG.clientset.InstanceGroupsFor(cluster).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	instanceGroups := make([]*kops.InstanceGroup, 0, len(list.Items))
	for i := range list.Items {
		instanceGroups = append(instanceGroups, &list.Items[i])
	}

	osASG.ApplyCmd = &cloudup.ApplyClusterCmd{
		Cloud:              osASG.Cloud,
		Clientset:          osASG.clientset,
		Cluster:            cluster,
		InstanceGroups:     instanceGroups,
		Phase:              "",
		TargetName:         cloudup.TargetDryRun,
		OutDir:             "out",
		AllowKopsDowngrade: true,
	}
	return nil
}

// DryRun scans do we need run update or not.
// Currently it supports scaling up and down instances.
// we do not use kops update cluster dryrun because it will make lots of API queries against OpenStack.
func (osASG *openstackASG) dryRun() (bool, error) {
	osCloud, ok := osASG.Cloud.(openstack.OpenstackCloud)
	if !ok {
		return false, fmt.Errorf("type assertion error")
	}

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
		if ok && ok2 && val == cluster.Name { //nolint: nestif
			maintenanceVal, ok3 := instance.Metadata["maintenance"]
			if instance.Status == "SHUTOFF" && (!ok3 || maintenanceVal != "true") {
				startErr := servers.Start(context.TODO(), osCloud.ComputeClient(), instance.ID).ExtractErr()
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
		if fi.ValueOf(ig.Spec.MinSize) < currentIGs[ig.Name] {
			glog.V(2).Infof("Scaling down running update --yes")
			return true, nil
		}
		if fi.ValueOf(ig.Spec.MinSize) > currentIGs[ig.Name] {
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
	if _, err := osASG.ApplyCmd.Run(ctx); err != nil {
		return err
	}
	return nil
}

func (osASG *openstackASG) enableMetrics(lbMetrics, quotaMetrics bool) error {
	ctx := context.Background()
	authOpts, err := openstackv2.AuthOptionsFromEnv()
	if err != nil {
		glog.Errorf("Error building auth options from env %v", err)
		return err
	}
	provider, err := openstackv2.AuthenticatedClient(ctx, authOpts)
	if err != nil {
		glog.Errorf("Error building openstack authenticated client %v", err)
		return err
	}

	if lbMetrics {
		client, err := openstackv2.NewLoadBalancerV2(provider, gophercloud.EndpointOpts{})
		if err != nil {
			glog.Errorf("Error building openstack load balancer client %v", err)
			return err
		}

		err = osASG.getLoadBalancerMetrics(ctx, client)
		if err != nil {
			return err
		}

		err = osASG.getMemberMetrics(ctx, client)
		if err != nil {
			return err
		}
	}

	if quotaMetrics {
		computeClient, err := openstackv2.NewComputeV2(provider, gophercloud.EndpointOpts{
			Type: "compute",
		})
		if err != nil {
			glog.Errorf("Error building openstack compute client %v", err)
			return err
		}

		err = osASG.getComputeQuota(ctx, computeClient, authOpts.TenantID, authOpts.TenantName)
		if err != nil {
			glog.Errorf("Error fetching compute quota %v", err)
			return err
		}

		volumeClient, err := openstackv2.NewBlockStorageV3(provider, gophercloud.EndpointOpts{
			Type: "volumev3",
		})
		if err != nil {
			glog.Errorf("Error building openstack volume client %v", err)
			return err
		}

		err = osASG.getVolumeQuotas(ctx, volumeClient, authOpts.TenantID, authOpts.TenantName)
		if err != nil {
			glog.Errorf("Error fetching volume quota %v", err)
		}
	}

	return nil
}

func (osASG *openstackASG) getComputeQuota(ctx context.Context, client *gophercloud.ServiceClient, tenantID string, tenantName string) error {
	quotaset, err := quotasets.GetDetail(ctx, client, tenantID).Extract()
	if err != nil {
		return err
	}

	ramUsed.WithLabelValues(tenantID, tenantName).Set(float64(quotaset.RAM.InUse + quotaset.RAM.Reserved))
	ramQuota.WithLabelValues(tenantID, tenantName).Set(float64(quotaset.RAM.Limit))

	secGroupsUsed.WithLabelValues(tenantID, tenantName).Set(float64(quotaset.SecurityGroups.InUse + quotaset.SecurityGroups.Reserved))
	secGroupsQuota.WithLabelValues(tenantID, tenantName).Set(float64(quotaset.SecurityGroups.Limit))

	coreUsed.WithLabelValues(tenantID, tenantName).Set(float64(quotaset.Cores.InUse + quotaset.Cores.Reserved))
	coreQuota.WithLabelValues(tenantID, tenantName).Set(float64(quotaset.Cores.Limit))

	instancesUsed.WithLabelValues(tenantID, tenantName).Set(float64(quotaset.Instances.InUse + quotaset.Instances.Reserved))
	instancesQuota.WithLabelValues(tenantID, tenantName).Set(float64(quotaset.Instances.Limit))

	serverGroupsUsed.WithLabelValues(tenantID, tenantName).Set(float64(quotaset.ServerGroups.InUse + quotaset.ServerGroups.Reserved))
	serverGroupsQuota.WithLabelValues(tenantID, tenantName).Set(float64(quotaset.ServerGroups.Limit))
	return nil
}

func (osASG *openstackASG) getVolumeQuotas(ctx context.Context, client *gophercloud.ServiceClient, tenantID string, tenantName string) error {
	quotaset, err := cinderquota.GetUsage(ctx, client, tenantID).Extract()
	if err != nil {
		return err
	}

	volumesUsed.WithLabelValues(tenantID, tenantName).Set(float64(quotaset.Volumes.InUse + quotaset.Volumes.Allocated + quotaset.Volumes.Reserved))
	volumesQuota.WithLabelValues(tenantID, tenantName).Set(float64(quotaset.Volumes.Limit))

	spaceUsed.WithLabelValues(tenantID, tenantName).Set(float64(quotaset.Gigabytes.InUse + quotaset.Gigabytes.Allocated + quotaset.Gigabytes.Reserved))
	spaceQuota.WithLabelValues(tenantID, tenantName).Set(float64(quotaset.Gigabytes.Limit))

	return nil
}

func (osASG *openstackASG) getLoadBalancerMetrics(ctx context.Context, client *gophercloud.ServiceClient) error {
	allPages, err := loadbalancers.List(client, loadbalancers.ListOpts{}).AllPages(ctx)
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
		stats, err := loadbalancers.GetStats(ctx, client, lb.ID).Extract()
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

func (osASG *openstackASG) getMemberMetrics(ctx context.Context, client *gophercloud.ServiceClient) error {
	allPages, err := pools.List(client, pools.ListOpts{}).AllPages(ctx)
	if err != nil {
		glog.Errorf("Error listing load balancer pool pages %v", err)
		return err
	}
	allPools, err := pools.ExtractPools(allPages)
	if err != nil {
		glog.Errorf("Error extracting load balancer pools %v", err)
		return err
	}

	lbPoolMember.Reset()
	for _, pool := range allPools {
		allPages, err := pools.ListMembers(client, pool.ID, pools.ListMembersOpts{}).AllPages(ctx)
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
			lbPoolMember.WithLabelValues(member.Name, member.ID, pool.Name, pool.ID, pool.Loadbalancers[0].ID,
				member.ProvisioningStatus, member.OperatingStatus, strconv.Itoa(member.Weight)).Set(float64(1))
		}
	}

	return nil
}
