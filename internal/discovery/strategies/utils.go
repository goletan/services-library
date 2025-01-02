package strategies

import (
	"github.com/goletan/services-library/shared/types"
	v1 "k8s.io/api/core/v1"
)

func ConvertPorts(ports []v1.ServicePort) []types.ServicePort {
	var converted []types.ServicePort
	for _, port := range ports {
		converted = append(converted, types.ServicePort{
			Name:     port.Name,
			Port:     int(port.Port),
			Protocol: string(port.Protocol),
		})
	}
	return converted
}

func MatchLabels(serviceLabels map[string]string, filterLabels map[string]string) bool {
	for key, value := range filterLabels {
		if serviceLabels[key] != value {
			return false
		}
	}
	return true
}

func MatchTags(serviceTags map[string]string, filterTags map[string]string) bool {
	for key, value := range filterTags {
		if serviceValue, exists := serviceTags[key]; !exists || serviceValue != value {
			return false
		}
	}
	return true
}
