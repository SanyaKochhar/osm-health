package smi

import (
	"testing"
	v1 "k8s.io/api/core/v1"
	split "github.com/servicemeshinterface/smi-sdk-go/pkg/apis/split/v1alpha2"
)

func TestIsInTrafficSplit(t *testing.T) {
	assert := tassert.New(t)

	type test struct {
		pod             v1.Pod
		services        []v1.Service
		trafficSplits   []*split.TrafficSplit
		isErrorExpected bool
	}

	testCases := []test{
		/**
		  expected nil:
		  		- pod has labels
		  		- service matching labels found
		  		- traffic split matching service backend found

		  errors:
		  		1. no services
		  		2. no traffic splits
		  		3. pod labels match service but service doesn't match ts backend

		  test for helper:
		  		1. error: no services
		  		2. nil: services match pod labels
				3. nil: no match - empty list
				4. nil: no labels/selectors - empty list
		*/
		{
			pod: v1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod-1",
					Labels: map[string]string{
						"app": "bookstore-v1",
					},
				},
			},
			services: []v1.Service{
				Name: "TODO"
			},
			trafficSplits: []*split.TrafficSplit{
			// TODO
			},
			isErrorExpected: false,
			},
		}
	}

	for _, tc := range testCases {
		trafficSplitChecker := IsInTrafficSplit(&tc.pod, 2)
// TODO
		assert.Equal(tc.expectedError, numContainersChecker.Run())
	}
}
