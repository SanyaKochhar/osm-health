package pod

import (
	"context"
	"log"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	testclient "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
)

// EnvoySidecarImageCheck
/**
- happy path: create a fake mesh config, check against meshconfig, all g
- sad path: create a fake mesh config with wrong value, no bueno
*/

func TestHasExpectedEnvoyImage(t *testing.T) {
	assert := tassert.New(t)
	meshConfigClientSet := testclient.NewSimpleClientset()
	stop := make(chan struct{})
	defer close(stop)
	osmNamespace := "osm-system"
	osmMeshConfigName := "osm-mesh-config"

	type test struct {
		pod            corev1.Pod
		meshConfigSpec *v1alpha1.MeshConfigSpec
		expectedError  error
	}

	testCases := []test{
		{
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod-1",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "EnvoyContainer",
							Image: "envoyproxy/envoy-alpine:v1.18.3",
						},
					},
				},
			},
			meshConfigSpec: &v1alpha1.MeshConfigSpec{
				Sidecar: v1alpha1.SidecarSpec{
					EnvoyImage:         "envoyproxy/envoy-alpine:v1.18.3",
					InitContainerImage: "openservicemesh/init:v0.0.0",
				},
			},
			expectedError: nil,
		},
	}

	for _, tc := range testCases {
		sidecarImageChecker := HasExpectedEnvoyImage(&tc.pod)
		meshConfig := v1alpha1.MeshConfig{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: osmNamespace,
				Name:      osmMeshConfigName,
			},
			Spec: *tc.meshConfigSpec,
		}
		if _, err := meshConfigClientSet.ConfigV1alpha1().MeshConfigs(osmNamespace).Create(context.TODO(), &meshConfig, metav1.CreateOptions{}); err != nil {
			log.Fatalf("[TEST] Error creating MeshConfig %s/%s/: %s", meshConfig.Namespace, meshConfig.Name, err.Error())
		}

		assert.Equal(tc.expectedError, sidecarImageChecker.Run())
	}
}

// MinNumContainersCheck
/**
- happy path: correct num containers, no error
- sad path:
	- pod with too few containers
	- invalid pod
*/
