package autoscaler

import (
	"fmt"
	"reflect"
	"time"

	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kops/pkg/apis/kops"
	"k8s.io/kops/pkg/client/simple"
	"k8s.io/kops/pkg/client/simple/vfsclientset"
	"k8s.io/kops/upup/pkg/fi"
	"k8s.io/kops/upup/pkg/fi/cloudup"
	"k8s.io/kops/util/pkg/reflectutils"
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
}

// Run will execute cluster check in loop periodically
func Run(opts *Options) error {
	registryBase, err := vfs.Context.BuildVfsPath(opts.StateStore)
	if err != nil {
		return fmt.Errorf("error parsing registry path %q: %v", opts.StateStore, err)
	}

	clientset := vfsclientset.NewVFSClientset(registryBase, true)
	osASG := &openstackASG{
		opts:      opts,
		clientset: clientset,
	}

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
	return nil
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

func (osASG *openstackASG) dryRun() (bool, error) {
	osASG.ApplyCmd.TargetName = cloudup.TargetDryRun
	osASG.ApplyCmd.DryRun = true

	if err := osASG.ApplyCmd.Run(); err != nil {
		return false, err
	}
	target := osASG.ApplyCmd.Target.(*fi.DryRunTarget)
	if target.HasChanges() {
		creates, updates := target.Changes()
		for k := range creates {
			// scale up
			if k == "Instance" {
				glog.Infof("Scaling up running update --yes")
				return true, nil
			}
		}

		for k, v := range updates {
			// scale down
			if k == "ServerGroup" {
				maxSizeChanged := false
				changes := reflect.ValueOf(v)
				if changes.Kind() == reflect.Ptr && !changes.IsNil() {
					changes = changes.Elem()
				}
				for i := 0; i < changes.NumField(); i++ {
					fieldValue := reflectutils.ValueAsString(changes.Field(i))
					if changes.Type().Field(i).Name == "MaxSize" && fieldValue != "" {
						maxSizeChanged = true
						break
					}
				}
				if maxSizeChanged {
					glog.Infof("Scaling down running update --yes")
				}
				return maxSizeChanged, nil
			}
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
