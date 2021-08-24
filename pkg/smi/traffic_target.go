package smi

import (
	"context"
	"fmt"
	"strings"

	access "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/access/v1alpha3"
	smiAccessClient "github.com/servicemeshinterface/smi-sdk-go/pkg/gen/client/access/clientset/versioned"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm-health/pkg/common"
	"github.com/openservicemesh/osm-health/pkg/common/outcomes"
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
	if check.cfg.IsPermissiveTrafficPolicyMode() {
		return outcomes.DiagnosticOutcome{LongDiagnostics: fmt.Sprintf("OSM is in permissive traffic policy modes -- all meshed pods can communicate and SMI access policies are not applicable")}
	}
	matchingTrafficTargets, err := getMatchingTrafficTargets(check.accessClient, check.srcPod, check.dstPod)
	if err != nil {
		return outcomes.FailedOutcome{Error: err}
	}
	if len(matchingTrafficTargets) == 0 {
		return outcomes.DiagnosticOutcome{LongDiagnostics: fmt.Sprintf("Pod '%s/%s' is not allowed to communicate to pod '%s/%s' via any SMI TrafficTarget policy\n",
			check.srcPod.Namespace, check.srcPod.Name, check.dstPod.Namespace, check.dstPod.Name)}
	}
	ret := fmt.Sprintf("Pod '%s/%s' is allowed to communicate to pod '%s/%s' via SMI TrafficTarget policy/policies:",
		check.srcPod.Namespace, check.srcPod.Name, check.dstPod.Namespace, check.dstPod.Name)
	for _, trafficTarget := range matchingTrafficTargets {
		ret = fmt.Sprintf("%s %s, ", ret, trafficTarget.Name)
	}
	return outcomes.DiagnosticOutcome{LongDiagnostics: strings.TrimSuffix(ret, ", ")}
}

// Suggestion implements common.Runnable
func (check TrafficTargetCheck) Suggestion() string {
	return fmt.Sprintf("Check that source and desintation pod are referred to in a TrafficTarget. To get relevant TrafficTargets, use: \"kubectl get traffictarget -n %s -o yaml\"", check.dstPod.Namespace)
}

// FixIt implements common.Runnable
func (check TrafficTargetCheck) FixIt() error {
	panic("implement me")
}

func getMatchingTrafficTargets(smiAccessClient smiAccessClient.Interface, srcPod *corev1.Pod, dstPod *corev1.Pod) ([]*access.TrafficTarget, error) {
	trafficTargets, err := smiAccessClient.AccessV1alpha3().TrafficTargets(dstPod.Namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	//var foundTrafficTarget bool
	var matchingTrafficTargets []*access.TrafficTarget
	for _, trafficTarget := range trafficTargets.Items {
		spec := trafficTarget.Spec

		// Map traffic targets to the given pods
		if !cli.DoesTargetRefDstPod(spec, dstPod) {
			continue
		}
		// The TrafficTarget destination is associated to 'dstPod', check if 'srcPod` is an allowed source to this destination
		if cli.DoesTargetRefSrcPod(spec, srcPod) {
			matchingTrafficTargets = append(matchingTrafficTargets, &trafficTarget)
		}
	}
	return matchingTrafficTargets, nil
}

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
	matchingTrafficTargets, err := getMatchingTrafficTargets(check.accessClient, check.srcPod, check.dstPod)
	if err != nil {
		return outcomes.FailedOutcome{Error: err}
	}
	if len(matchingTrafficTargets) == 0 {
		return outcomes.DiagnosticOutcome{LongDiagnostics: fmt.Sprintf("Pod '%s/%s' is not allowed to communicate to pod '%s/%s' via any SMI TrafficTarget policy\n",
			check.srcPod.Namespace, check.srcPod.Name, check.dstPod.Namespace, check.dstPod.Name)}
	}
	unsupportedRouteTargets := map[string]string{}
	for _, trafficTarget := range matchingTrafficTargets {
		spec := trafficTarget.Spec
		for _, rule := range spec.Rules {
			kind := rule.Kind
			if !(kind == HTTPRouteGroupKind || kind == TCPRouteKind) {
				unsupportedRouteTargets[trafficTarget.Name] = kind
			}
		}
	}
	if len(unsupportedRouteTargets) > 0 {
		errorString := check.generateErrorMessage(unsupportedRouteTargets)
		return outcomes.FailedOutcome{Error: fmt.Errorf(errorString)}
	}
	return outcomes.SuccessfulOutcomeWithoutDiagnostics{}
}

func (c *RoutesValidityCheck) generateErrorMessage(targetToKindMap map[string]string) string {
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
