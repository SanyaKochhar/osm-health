package podhelper

import (
	"context"
	"log"
	"testing"

	tassert "github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/openservicemesh/osm/pkg/apis/config/v1alpha1"
	"github.com/openservicemesh/osm/pkg/configurator"
	testclient "github.com/openservicemesh/osm/pkg/gen/client/config/clientset/versioned/fake"
)

func TestHasExpectedNumContainers(t *testing.T) {
	assert := tassert.New(t)

	type test struct {
		pod           corev1.Pod
		expectedError error
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
							Image: "envoyproxy/envoy-alpine:v1.18.777",
						},
						{
							Name:  "AppContainer",
							Image: "random/app:v0.0.0",
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:  "OsmInit",
							Image: "openservicemesh/init:v0.0.0",
						},
					},
				},
			},
			expectedError: nil,
		},
		{
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod-2",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "EnvoyContainer",
							Image: "envoyproxy/envoy-alpine:v1.18.555",
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:  "OsmInit",
							Image: "openservicemesh/init:v0.0.0",
						},
					},
				},
			},
			expectedError: ErrExpectedMinNumContainers,
		},
		{
			pod: corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod-3",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "EnvoyContainer",
							Image: "envoyproxy/envoy-alpine:v1.18.555",
						},
						{
							Name:  "AppContainer",
							Image: "random/app:v0.0.0",
						},
					},
				},
			},
			expectedError: ErrExpectedMinNumContainers,
		},
	}

	for _, tc := range testCases {
		// TODO: change to 2 once HasOsmInitCheck is added
		numContainersChecker := HasMinExpectedContainers(&tc.pod, 3)

		assert.Equal(tc.expectedError, numContainersChecker.Run())
	}
}

func TestHasExpectedOsmInitImage(t *testing.T) {
	assert := tassert.New(t)
	osmNamespace := "osm-system"
	osmMeshConfigName := "test-mesh-config"

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
							Name:  "AppContainer",
							Image: "randomimage/random:v0.0.0",
						},
					},
					InitContainers: []corev1.Container{
						{
							Name:  "OsmInit",
							Image: "openservicemesh/TEST:v0.9.1",
						},
					},
				},
			},
			meshConfigSpec: &v1alpha1.MeshConfigSpec{
				Sidecar: v1alpha1.SidecarSpec{
					InitContainerImage: "openservicemesh/TEST:v0.9.1",
				},
			},
			expectedError: nil,
		},
		//{
		//	pod: corev1.Pod{
		//		ObjectMeta: metav1.ObjectMeta{
		//			Name: "pod-2",
		//		},
		//		Spec: corev1.PodSpec{
		//			Containers: []corev1.Container{
		//				{
		//					Name:  "EnvoyContainer",
		//					Image: "envoyproxy/envoy-alpine:v1.18.555",
		//				},
		//			},
		//		},
		//	},
		//	meshConfigSpec: &v1alpha1.MeshConfigSpec{
		//		Sidecar: v1alpha1.SidecarSpec{
		//			EnvoyImage:         "randomimage/random:v0.0.0",
		//			InitContainerImage: "openservicemesh/init:v0.0.0",
		//		},
		//	},
		//	expectedError: nil,
		//},
	}

	for _, tc := range testCases {
		meshConfigClientSet := testclient.NewSimpleClientset()
		stop := make(chan struct{})
		defer close(stop)
		//configurator := kuberneteshelper.GetOsmConfigurator(common.MeshNamespace(osmNamespace), osmMeshConfigName)
		cfg := configurator.NewConfigurator(meshConfigClientSet, stop, osmNamespace, osmMeshConfigName)
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
		jsonmc, _ := cfg.GetMeshConfigJSON()
		log.Printf("Meshconfig created %s", jsonmc)
		sidecarImageChecker := HasExpectedOsmInitImage(cfg, &tc.pod)

		assert.Equal(tc.expectedError, sidecarImageChecker.Run())
	}
}
