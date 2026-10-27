package service

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"dodevops-api/api/k8s/model"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/validation"
)

func validateScalingTarget(ref autoscalingv2.CrossVersionObjectReference, vertical bool) error {
	if ref.APIVersion != "apps/v1" || (ref.Kind != "Deployment" && ref.Kind != "StatefulSet" && !(vertical && ref.Kind == "DaemonSet")) {
		return fmt.Errorf("目标必须为 apps/v1 Deployment、StatefulSet；VPA 也支持 DaemonSet")
	}
	if len(validation.IsDNS1123Subdomain(ref.Name)) > 0 {
		return fmt.Errorf("目标名称无效")
	}
	return nil
}
func validateHPA(s *autoscalingv2.HorizontalPodAutoscalerSpec) error {
	if s == nil {
		return fmt.Errorf("缺少 hpa 配置")
	}
	if err := validateScalingTarget(s.ScaleTargetRef, false); err != nil {
		return err
	}
	if s.MinReplicas == nil || *s.MinReplicas < 1 || s.MaxReplicas < *s.MinReplicas {
		return fmt.Errorf("HPA 副本边界必须满足 1 <= minReplicas <= maxReplicas")
	}
	if len(s.Metrics) == 0 {
		return fmt.Errorf("至少配置一个伸缩指标")
	}
	for _, m := range s.Metrics {
		count := 0
		for _, present := range []bool{m.Resource != nil, m.ContainerResource != nil, m.Pods != nil, m.Object != nil, m.External != nil} {
			if present {
				count++
			}
		}
		if count != 1 {
			return fmt.Errorf("每项指标必须且只能指定一个指标来源")
		}
		var target autoscalingv2.MetricTarget
		switch m.Type {
		case autoscalingv2.ResourceMetricSourceType:
			if m.Resource == nil {
				return fmt.Errorf("Resource 指标来源缺失")
			}
			target = m.Resource.Target
			if m.Resource.Name != "cpu" && m.Resource.Name != "memory" {
				return fmt.Errorf("Resource 指标仅支持 cpu 或 memory")
			}
		case autoscalingv2.ContainerResourceMetricSourceType:
			if m.ContainerResource == nil || m.ContainerResource.Container == "" {
				return fmt.Errorf("ContainerResource 必须指定容器")
			}
			target = m.ContainerResource.Target
			if m.ContainerResource.Name != "cpu" && m.ContainerResource.Name != "memory" {
				return fmt.Errorf("ContainerResource 指标仅支持 cpu 或 memory")
			}
		case autoscalingv2.PodsMetricSourceType:
			if m.Pods == nil || m.Pods.Metric.Name == "" {
				return fmt.Errorf("Pods 指标必须指定名称")
			}
			target = m.Pods.Target
			if target.Type != autoscalingv2.AverageValueMetricType {
				return fmt.Errorf("Pods 指标必须使用 AverageValue")
			}
		case autoscalingv2.ObjectMetricSourceType:
			if m.Object == nil || m.Object.Metric.Name == "" || m.Object.DescribedObject.Name == "" || m.Object.DescribedObject.Kind == "" || m.Object.DescribedObject.APIVersion == "" {
				return fmt.Errorf("Object 指标必须指定名称和 describedObject")
			}
			target = m.Object.Target
			if target.Type == autoscalingv2.UtilizationMetricType {
				return fmt.Errorf("Object 指标不支持 Utilization")
			}
		case autoscalingv2.ExternalMetricSourceType:
			if m.External == nil || m.External.Metric.Name == "" {
				return fmt.Errorf("External 指标必须指定名称")
			}
			target = m.External.Target
			if target.Type == autoscalingv2.UtilizationMetricType {
				return fmt.Errorf("External 指标不支持 Utilization")
			}
		default:
			return fmt.Errorf("未知的 HPA 指标类型")
		}
		count = 0
		for _, p := range []bool{target.Value != nil, target.AverageValue != nil, target.AverageUtilization != nil} {
			if p {
				count++
			}
		}
		if count != 1 {
			return fmt.Errorf("每个指标目标必须且只能指定一个数值")
		}
		switch target.Type {
		case autoscalingv2.UtilizationMetricType:
			if target.AverageUtilization == nil || *target.AverageUtilization <= 0 {
				return fmt.Errorf("利用率目标必须大于 0")
			}
		case autoscalingv2.AverageValueMetricType:
			if target.AverageValue == nil || target.AverageValue.Sign() <= 0 {
				return fmt.Errorf("AverageValue 必须大于 0")
			}
		case autoscalingv2.ValueMetricType:
			if m.Type == autoscalingv2.ResourceMetricSourceType || m.Type == autoscalingv2.ContainerResourceMetricSourceType {
				return fmt.Errorf("资源指标不支持 Value")
			}
			if target.Value == nil || target.Value.Sign() <= 0 {
				return fmt.Errorf("Value 必须大于 0")
			}
		default:
			return fmt.Errorf("未知的指标目标类型")
		}
	}
	if s.Behavior != nil {
		for _, b := range []*autoscalingv2.HPAScalingRules{s.Behavior.ScaleUp, s.Behavior.ScaleDown} {
			if b == nil {
				continue
			}
			if b.StabilizationWindowSeconds != nil && (*b.StabilizationWindowSeconds < 0 || *b.StabilizationWindowSeconds > 3600) {
				return fmt.Errorf("稳定窗口必须为 0–3600 秒")
			}
			if b.SelectPolicy != nil && *b.SelectPolicy != "Max" && *b.SelectPolicy != "Min" && *b.SelectPolicy != "Disabled" {
				return fmt.Errorf("selectPolicy 必须为 Max、Min 或 Disabled")
			}
			for _, p := range b.Policies {
				if (p.Type != "Pods" && p.Type != "Percent") || p.Value <= 0 || p.PeriodSeconds < 1 || p.PeriodSeconds > 1800 {
					return fmt.Errorf("伸缩策略需使用 Pods/Percent、正数 value 及 1–1800 秒周期")
				}
			}
		}
	}
	return nil
}
func validateVPA(s *model.VPASpec) error {
	if s == nil {
		return fmt.Errorf("缺少 vpa 配置")
	}
	if err := validateScalingTarget(s.TargetRef, true); err != nil {
		return err
	}
	switch s.UpdatePolicy.UpdateMode {
	case "", "Off", "Initial", "Recreate", "Auto", "InPlaceOrRecreate", "InPlace":
	default:
		return fmt.Errorf("无效的 VPA updateMode")
	}
	if s.UpdatePolicy.MinReplicas != nil && *s.UpdatePolicy.MinReplicas < 1 {
		return fmt.Errorf("minReplicas 必须至少为 1")
	}
	names := map[string]bool{}
	for _, p := range s.ResourcePolicy.ContainerPolicies {
		if (p.ContainerName != "*" && len(validation.IsDNS1123Label(p.ContainerName)) > 0) || names[p.ContainerName] {
			return fmt.Errorf("容器名称无效或重复")
		}
		names[p.ContainerName] = true
		if p.Mode != "" && p.Mode != "Auto" && p.Mode != "Off" {
			return fmt.Errorf("容器 mode 必须为 Auto 或 Off")
		}
		if p.ControlledValues != "" && p.ControlledValues != "RequestsOnly" && p.ControlledValues != "RequestsAndLimits" {
			return fmt.Errorf("controlledValues 必须为 RequestsOnly 或 RequestsAndLimits")
		}
		for _, r := range p.ControlledResources {
			if r != "cpu" && r != "memory" {
				return fmt.Errorf("controlledResources 仅支持 cpu、memory")
			}
		}
		for _, bounds := range []map[string]string{p.MinAllowed, p.MaxAllowed} {
			for k, v := range bounds {
				q, err := resource.ParseQuantity(v)
				if (k != "cpu" && k != "memory") || err != nil || q.Sign() < 0 {
					return fmt.Errorf("资源边界必须为非负的 CPU/内存 quantity")
				}
			}
		}
		for k, v := range p.MinAllowed {
			if maximum, ok := p.MaxAllowed[k]; ok {
				min, _ := resource.ParseQuantity(v)
				max, _ := resource.ParseQuantity(maximum)
				if min.Cmp(max) > 0 {
					return fmt.Errorf("%s 的 minAllowed 不能大于 maxAllowed", k)
				}
			}
		}
	}
	return nil
}

const kpaPrefix = "autoscaling.knative.dev/"

var managedKPAKeys = []string{"class", "metric", "target", "min-scale", "max-scale", "window", "panic-window-percentage", "panic-threshold-percentage", "target-utilization-percentage", "scale-down-delay"}

func kpaAnnotations(c *model.KPAConfig) (map[string]string, error) {
	if c == nil {
		return nil, fmt.Errorf("缺少 kpa 配置")
	}
	if c.Metric != "" && c.Metric != "concurrency" && c.Metric != "rps" {
		return nil, fmt.Errorf("KPA metric 必须为 concurrency 或 rps")
	}
	result := map[string]string{kpaPrefix + "class": "kpa.autoscaling.knative.dev"}
	if c.Metric != "" {
		result[kpaPrefix+"metric"] = c.Metric
	}
	floats := []struct {
		key          string
		value        *float64
		min, max     float64
		exclusiveMin bool
	}{
		{"target", c.Target, 0, math.MaxFloat64, true},
		{"panic-window-percentage", c.PanicWindowPercentage, 1, 100, false},
		{"panic-threshold-percentage", c.PanicThresholdPercentage, 110, 1000, false},
		{"target-utilization-percentage", c.TargetUtilizationPercentage, 0, 100, true},
	}
	for _, f := range floats {
		if f.value == nil {
			continue
		}
		n := *f.value
		if math.IsNaN(n) || math.IsInf(n, 0) || n < f.min || n > f.max || (f.exclusiveMin && n == f.min) {
			return nil, fmt.Errorf("KPA %s 数值越界", f.key)
		}
		result[kpaPrefix+f.key] = strconv.FormatFloat(n, 'f', -1, 64)
	}
	integers := []struct {
		key      string
		value    *int64
		min, max int64
		duration bool
	}{
		{"min-scale", c.MinScale, 0, math.MaxInt32, false},
		{"max-scale", c.MaxScale, 0, math.MaxInt32, false},
		{"window", c.WindowSeconds, 6, 3600, true},
		{"scale-down-delay", c.ScaleDownDelaySeconds, 0, 3600, true},
	}
	for _, f := range integers {
		if f.value == nil {
			continue
		}
		n := *f.value
		if n < f.min || n > f.max {
			return nil, fmt.Errorf("KPA %s 必须为 %d–%d", f.key, f.min, f.max)
		}
		value := strconv.FormatInt(n, 10)
		if f.duration {
			value += "s"
		}
		result[kpaPrefix+f.key] = value
	}
	return result, nil
}
func validateKPABounds(annotations map[string]string) error {
	minimum, err := strconv.ParseInt(annotations[kpaPrefix+"min-scale"], 10, 64)
	if err != nil && annotations[kpaPrefix+"min-scale"] != "" {
		return fmt.Errorf("现有 min-scale 无效")
	}
	maximum, err := strconv.ParseInt(annotations[kpaPrefix+"max-scale"], 10, 64)
	if err != nil && annotations[kpaPrefix+"max-scale"] != "" {
		return fmt.Errorf("现有 max-scale 无效")
	}
	if maximum > 0 && maximum < minimum {
		return fmt.Errorf("maxScale 为 0 表示无限制，否则必须 >= minScale")
	}
	return nil
}
func hpaUsesResourceMetrics(spec map[string]interface{}) bool {
	metrics, _ := spec["metrics"].([]interface{})
	for _, v := range metrics {
		m, _ := v.(map[string]interface{})
		t, _ := m["type"].(string)
		if t == "Resource" || t == "ContainerResource" {
			return true
		}
	}
	return false
}
func scalingTargetKey(ref map[string]interface{}) string {
	version, _ := ref["apiVersion"].(string)
	kind, _ := ref["kind"].(string)
	name, _ := ref["name"].(string)
	return strings.Split(version, "/")[0] + "/" + kind + "/" + name
}
