package autoscaler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	// Import pprof package, needed for debugging.
	_ "net/http/pprof" //nolint: gosec

	"github.com/golang/glog"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
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
	LogLevel       string
	Sleep          int
	StateStore     string
	AccessKey      string
	SecretKey      string
	CustomEndpoint string
	ClusterName    string
}

type openstackASG struct {
	ApplyCmd  *cloudup.ApplyClusterCmd
	clientset simple.Clientset
	opts      *Options
	Cloud     fi.Cloud
}

// Run will execute cluster check in loop periodically.
func Run(opts *Options) error {
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
