package smi

import (
	"context"
	"fmt"

	"github.com/openservicemesh/osm-health/pkg/common/outcomes"

	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiAccessClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm-health/pkg/common"
	"github.com/openservicemesh/osm/pkg/cli"
	"github.com/openservicemesh/osm/pkg/configurator"
)

// Verify interface compliance
var _ common.Runnable = (*TrafficTargetCheck)(nil)

// TrafficTargetCheck implements common.Runnable
type TrafficTargetCheck struct {
	cfg          configurator.Configurator
	srcPod       *corev1.Pod
	dstPod       *corev1.Pod
	accessClient smiAccessClient.Interface
}

// IsInTrafficTarget checks whether the src and dest pods are referenced as src and dest in a TrafficTarget (in that order)
func IsInTrafficTarget(osmConfigurator configurator.Configurator, srcPod *corev1.Pod, dstPod *corev1.Pod, smiAccessClient smiAccessClient.Interface) TrafficTargetCheck {
	return TrafficTargetCheck{
		cfg:          osmConfigurator,
		srcPod:       srcPod,
		dstPod:       dstPod,
		accessClient: smiAccessClient,
	}
}

// Info implements common.Runnable
func (check TrafficTargetCheck) Description() string {
	return fmt.Sprintf("Checking whether there is a Traffic Target with source pod %s and destination pod %s", check.srcPod.Name, check.dstPod.Name)
}

// Run implements common.Runnable
func (check TrafficTargetCheck) Run() outcomes.Outcome {
	// Check if permissive mode is enabled, in which case every meshed pod is allowed to communicate with each other
	// TODO: this should not be an error! ref issue #63. Change to diagnostic once #70 is merged
	if check.cfg.IsPermissiveTrafficPolicyMode() {
		return outcomes.DiagnosticOutcome{LongDiagnostics: fmt.Sprintf("OSM is in permissive traffic policy modes -- all meshed pods can communicate and SMI access policies are not applicable")}
	}
	trafficTargets, err := check.accessClient.AccessV1alpha3().TrafficTargets(check.dstPod.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return outcomes.FailedOutcome{Error: err}
	}
	for _, trafficTarget := range trafficTargets.Items {
		if doesTargetMatchPods(trafficTarget.Spec, check.srcPod, check.dstPod) {
			return outcomes.DiagnosticOutcome{LongDiagnostics: fmt.Sprintf("Pod '%s/%s' is allowed to communicate to pod '%s/%s' via the SMI TrafficTarget policy %q in namespace %s\n",
				check.srcPod.Namespace, check.srcPod.Name, check.dstPod.Namespace, check.dstPod.Name, trafficTarget.Name, trafficTarget.Namespace)}
		}
	}
	//trafficTargets, err := getTrafficTargets(check.accessClient, check.srcPod, check.dstPod)
	//if err != nil {
	//	return err
	//}
	//if len(trafficTargets) == 0 {
	//	return ErrPodsNotInTrafficTarget
	//}
	//return nil
	return outcomes.DiagnosticOutcome{LongDiagnostics: fmt.Sprintf("Pod '%s/%s' is not allowed to communicate to pod '%s/%s' via any SMI TrafficTarget policy\n",
		check.srcPod.Namespace, check.srcPod.Name, check.dstPod.Namespace, check.dstPod.Name)}
}

// Suggestion implements common.Runnable
func (check TrafficTargetCheck) Suggestion() string {
	return fmt.Sprintf("Check that source and desintation pod are referred to in a TrafficTarget. To get relevant TrafficTargets, use: \"kubectl get traffictarget -n %s -o yaml\"", check.dstPod.Namespace)
}

// FixIt implements common.Runnable
func (check TrafficTargetCheck) FixIt() error {
	panic("implement me")
}

func doesTargetMatchPods(spec access.TrafficTargetSpec, srcPod *corev1.Pod, dstPod *corev1.Pod) bool {
	// Map traffic targets to the given pods
	if !cli.DoesTargetRefDstPod(spec, dstPod) {
		return false
	}
	// The TrafficTarget destination is associated to 'dstPod', check if 'srcPod` is an allowed source to this destination
	if cli.DoesTargetRefSrcPod(spec, srcPod) {
		return true
	}
	return false
}

//
//func getTrafficTargets(smiAccessClient smiAccessClient.Interface, srcPod *corev1.Pod, dstPod *corev1.Pod) ([]*access.TrafficTarget, error) {
//	trafficTargets, err := smiAccessClient.AccessV1alpha3().TrafficTargets(dstPod.Namespace).List(context.TODO(), metav1.ListOptions{})
//	if err != nil {
//		return nil, err
//	}
//	//var foundTrafficTarget bool
//	var matchingTrafficTargets []*access.TrafficTarget
//	for _, trafficTarget := range trafficTargets.Items {
//		spec := trafficTarget.Spec
//
//		// Map traffic targets to the given pods
//		if !cli.DoesTargetRefDstPod(spec, check.dstPod) {
//			continue
//		}
//		// The TrafficTarget destination is associated to 'dstPod', check if 'srcPod` is an allowed source to this destination
//		if cli.DoesTargetRefSrcPod(spec, check.srcPod) {
//			//foundTrafficTarget = true
//			matchingTrafficTargets = matchingTrafficTargets.append(matchingTrafficTargets, &trafficTarget)
//		}
//	}
//	return matchingTrafficTargets, nil
//}

/**
way one: call the function on each target
a)
- iterate over all the traffic targets
- check if each one a match - return nil
- if no match found, return error

b) iterate over all the traffic targets
- check if it's a match
- if kind is a valid route, return nil
	- if not, return error
- return outcome???

way two: get the actual traffictargets
a) call the function, if error return error; if len(nil) return error. else return tt exists

b) call the function, returns list of traffic targets.
- iterate over each to see if valid
- if even one not valid, append the traffic target name to an existing string that will be returned as an error.
*/

// Verify interface compliance
var _ common.Runnable = (*RoutesValidityCheck)(nil)

// TrafficTargetCheck implements common.Runnable
type RoutesValidityCheck struct {
	cfg          configurator.Configurator
	srcPod       *corev1.Pod
	dstPod       *corev1.Pod
	accessClient smiAccessClient.Interface
}

// IsInTrafficTarget checks whether the src and dest pods are referenced as src and dest in a TrafficTarget (in that order)
func AreTrafficRoutesValid(osmConfigurator configurator.Configurator, srcPod *corev1.Pod, dstPod *corev1.Pod, smiAccessClient smiAccessClient.Interface) RoutesValidityCheck {
	return RoutesValidityCheck{
		cfg:          osmConfigurator,
		srcPod:       srcPod,
		dstPod:       dstPod,
		accessClient: smiAccessClient,
	}
}

const (
	HTTPRouteGroupKind = "HTTPRouteGroup"
	TCPRouteKind       = "TCPRoute"
)

// Info implements common.Runnable
func (check RoutesValidityCheck) Description() string {
	return fmt.Sprintf("Checking whether Traffic Targets with source pod %s and destination pod %s have valid routes (%s or %s)", check.srcPod.Name, check.dstPod.Name, HTTPRouteGroupKind, TCPRouteKind)
}

// Run implements common.Runnable
func (check RoutesValidityCheck) Run() outcomes.Outcome {
	// Check if permissive mode is enabled, in which case every meshed pod is allowed to communicate with each other
	if check.cfg.IsPermissiveTrafficPolicyMode() {
		return outcomes.DiagnosticOutcome{LongDiagnostics: fmt.Sprintf("OSM is in permissive traffic policy modes -- all meshed pods can communicate and SMI access policies are not applicable")}
	}
	trafficTargets, err := check.accessClient.AccessV1alpha3().TrafficTargets(check.dstPod.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return outcomes.FailedOutcome{Error: err}
	}
	unsupportedRouteTargets := map[string]string{}
	for _, trafficTarget := range trafficTargets.Items {
		spec := trafficTarget.Spec
		if !doesTargetMatchPods(spec, check.srcPod, check.dstPod) {
			continue
		}
		for _, rule := range spec.Rules {
			kind := rule.Kind
			if !(kind == HTTPRouteGroupKind || kind == TCPRouteKind) {
				unsupportedRouteTargets[trafficTarget.Name] = kind
			}
		}
	}
	if len(unsupportedRouteTargets) > 0 {
		errorString := generateErrorMessage(unsupportedRouteTargets)
		return outcomes.FailedOutcome{Error: fmt.Errorf(errorString)}
	}
	return outcomes.SuccessfulOutcomeWithoutDiagnostics{}
}

func generateErrorMessage(targetToKindMap map[string]string) string {
	errorString := fmt.Sprintf("Expected routes of kind %s or %s, found the following TrafficTargets with unsupported routes: \n", HTTPRouteGroupKind, TCPRouteKind)
	for target, kind := range targetToKindMap {
		errorString = fmt.Sprintf("%s %s: %s\n", errorString, target, kind)
	}
	return errorString
}

// Suggestion implements common.Runnable
func (check RoutesValidityCheck) Suggestion() string {
	return fmt.Sprintf("Check that TrafficTargets routes are of kind %s or %s. To get relevant TrafficTargets, use: \"kubectl get traffictarget -n %s -o yaml\"", HTTPRouteGroupKind, TCPRouteKind, check.dstPod.Namespace)
}

// FixIt implements common.Runnable
func (check RoutesValidityCheck) FixIt() error {
	panic("implement me")
}
