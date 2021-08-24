package smi

import (
	"context"
	"fmt"

	smiAccessClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/openservicemesh/osm-health/pkg/common"
	"github.com/openservicemesh/osm/pkg/cli"
	"github.com/openservicemesh/osm/pkg/configurator"
)

// Verify interface compliance
var _ common.Runnable = (*TrafficTargetCheck)(nil)

// TrafficTargetCheck implements common.Runnable
type TrafficTargetCheck struct {
	client       kubernetes.Interface
	cfg          configurator.Configurator
	srcPod       *corev1.Pod
	dstPod       *corev1.Pod
	accessClient smiAccessClient.Interface
}

// IsInTrafficTarget checks whether the src and dest pods are referenced as src and dest in a TrafficTarget (in that order)
func IsInTrafficTarget(client kubernetes.Interface, osmConfigurator configurator.Configurator, srcPod *corev1.Pod, dstPod *corev1.Pod, smiAccessClient smiAccessClient.Interface) TrafficTargetCheck {
	return TrafficTargetCheck{
		client:       client,
		cfg:          osmConfigurator,
		srcPod:       srcPod,
		dstPod:       dstPod,
		accessClient: smiAccessClient,
	}
}

// Info implements common.Runnable
func (check TrafficTargetCheck) Info() string {
	return fmt.Sprintf("Checking whether there is a Traffic Target with source pod %s and destination pod %s", check.srcPod.Name, check.dstPod.Name)
}

// Run implements common.Runnable
func (check TrafficTargetCheck) Run() error {
	// Check if permissive mode is enabled, in which case every meshed pod is allowed to communicate with each other
	// TODO: this should not be an error! ref issue #63. Change to diagnostic once #70 is merged
	if check.cfg.IsPermissiveTrafficPolicyMode() {
		return ErrPermissiveModeEnabled
	}
	trafficTargets, err := check.accessClient.AccessV1alpha3().TrafficTargets(check.dstPod.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, trafficTarget := range trafficTargets.Items {
		spec := trafficTarget.Spec

		// Map traffic targets to the given pods
		if !cli.DoesTargetRefDstPod(spec, check.dstPod) {
			continue
		}
		// The TrafficTarget destination is associated to 'dstPod', check if 'srcPod` is an allowed source to this destination
		if cli.DoesTargetRefSrcPod(spec, check.srcPod) {
			return nil
		}
	}
	return ErrPodsNotInTrafficTarget
}

// Suggestion implements common.Runnable
func (check TrafficTargetCheck) Suggestion() string {
	return fmt.Sprintf("Check that source and desintation pod are referred to in a TrafficTarget. To get relevant TrafficTargets, use: \"kubectl get traffictarget -n %s -o yaml\"", check.dstPod.Namespace)
}

// FixIt implements common.Runnable
func (check TrafficTargetCheck) FixIt() error {
	panic("implement me")
}
