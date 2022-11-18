package metrics

import (
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

// Client is a client for managing VolumeAttachments
type Client struct {
	Interface metricsclientset.Interface
	Namespace string
	Timeout   int
}
