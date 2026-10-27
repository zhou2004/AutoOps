package model

import autoscalingv2 "k8s.io/api/autoscaling/v2"

// AutoscalingRequest edits only the relevant policy, never arbitrary metadata or status.
type AutoscalingRequest struct {
	Name            string                                     `json:"name"`
	ResourceVersion string                                     `json:"resourceVersion"`
	HPA             *autoscalingv2.HorizontalPodAutoscalerSpec `json:"hpa,omitempty"`
	VPA             *VPASpec                                   `json:"vpa,omitempty"`
	KPA             *KPAConfig                                 `json:"kpa,omitempty"`
}
type VPASpec struct {
	TargetRef      autoscalingv2.CrossVersionObjectReference `json:"targetRef"`
	UpdatePolicy   VPAUpdatePolicy                           `json:"updatePolicy"`
	ResourcePolicy VPAResourcePolicy                         `json:"resourcePolicy"`
}
type VPAUpdatePolicy struct {
	UpdateMode  string `json:"updateMode,omitempty"`
	MinReplicas *int32 `json:"minReplicas,omitempty"`
}
type VPAResourcePolicy struct {
	ContainerPolicies []VPAContainerPolicy `json:"containerPolicies"`
}
type VPAContainerPolicy struct {
	ContainerName       string            `json:"containerName"`
	Mode                string            `json:"mode,omitempty"`
	MinAllowed          map[string]string `json:"minAllowed,omitempty"`
	MaxAllowed          map[string]string `json:"maxAllowed,omitempty"`
	ControlledResources []string          `json:"controlledResources,omitempty"`
	ControlledValues    string            `json:"controlledValues,omitempty"`
}
type KPAConfig struct {
	Metric                      string   `json:"metric,omitempty"`
	Target                      *float64 `json:"target,omitempty"`
	MinScale                    *int64   `json:"minScale,omitempty"`
	MaxScale                    *int64   `json:"maxScale,omitempty"`
	WindowSeconds               *int64   `json:"windowSeconds,omitempty"`
	PanicWindowPercentage       *float64 `json:"panicWindowPercentage,omitempty"`
	PanicThresholdPercentage    *float64 `json:"panicThresholdPercentage,omitempty"`
	TargetUtilizationPercentage *float64 `json:"targetUtilizationPercentage,omitempty"`
	ScaleDownDelaySeconds       *int64   `json:"scaleDownDelaySeconds,omitempty"`
}
