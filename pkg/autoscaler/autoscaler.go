package autoscaler

import (
	"fmt"
	"log"
	"net/http"
	"time"
	// import pprof package, needed for debugging
	_ "net/http/pprof"

	"github.com/golang/glog"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
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
	Cloud     openstack.OpenstackCloud
}

var (
	osInstances = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "openstack_instance",
			Help: "Openstack instance",
		},
		[]string{"name", "id", "status"},
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

	clientset := vfsclientset.NewVFSClientset(registryBase, true)
	osASG := &openstackASG{
		opts:      opts,
		clientset: clientset,
	}

	http.Handle("/metrics", promhttp.Handler())
	go http.ListenAndServe(":2112", nil)

	fails := 0
	for {
		if fails > 5 {
			return fmt.Errorf("Too many failed attempts")
		}
		time.Sleep(time.Duration(opts.Sleep) * time.Second)
		glog.Infof("Executing...\n")

		err := osASG.updateApplyCmd()
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
			err = osASG.update()
			if err != nil {
				glog.Errorf("Error updating cluster %v", err)
				fails++
				continue
			}
		}
		fails = 0
	}
}

func (osASG *openstackASG) updateApplyCmd() error {
	cluster, err := osASG.clientset.GetCluster(osASG.opts.ClusterName)
	if err != nil {
		return fmt.Errorf("error initializing cluster %v", err)
	}

	list, err := osASG.clientset.InstanceGroupsFor(cluster).List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	var instanceGroups []*kops.InstanceGroup
	for i := range list.Items {
		instanceGroups = append(instanceGroups, &list.Items[i])
	}

	if osASG.Cloud == nil {
		cloud, err := cloudup.BuildCloud(cluster)
		if err != nil {
			return err
		}
		osASG.Cloud = cloud.(openstack.OpenstackCloud)
	}

	osASG.ApplyCmd = &cloudup.ApplyClusterCmd{
		Clientset:      osASG.clientset,
		Cluster:        cluster,
		InstanceGroups: instanceGroups,
		Phase:          "",
		TargetName:     cloudup.TargetDryRun,
		OutDir:         "out",
		Models:         []string{"proto", "cloudup"},
	}
	return nil
}

// dryRun scans do we need run update or not
// currently it supports scaling up and down instances
// we do not use kops update cluster dryrun because it will make lots of API queries against OpenStack.
func (osASG *openstackASG) dryRun() (bool, error) {
	instances, err := osASG.Cloud.ListInstances(servers.ListOpts{})
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
			glog.Infof("Scaling down running update --yes")
			return true, nil
		}
		if fi.Int32Value(ig.Spec.MinSize) > currentIGs[ig.Name] {
			glog.Infof("Scaling up running update --yes")
			return true, nil
		}
	}
	return false, nil
}

func (osASG *openstackASG) update() error {
	osASG.ApplyCmd.TargetName = cloudup.TargetDirect
	osASG.ApplyCmd.DryRun = false
	var options fi.RunTasksOptions
	options.InitDefaults()
	osASG.ApplyCmd.RunTasksOptions = &options
	if err := osASG.ApplyCmd.Run(); err != nil {
		return err
	}
	return nil
}
