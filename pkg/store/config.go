package store

import (
	"fmt"

	"github.com/magicsong/okg-sidecar/pkg/expression"
	corev1 "k8s.io/api/core/v1"
)

type StorageType string

const (
	StorageTypePodAnnotation StorageType = "PodAnnotation"
	StorageTypeCRD           StorageType = "CRD"
	StorageTypeHTTPMetric    StorageType = "HTTPMetric"
)

type PodAnnotationConfig struct {
	AnnotationKey string `json:"annotationKey"` // Pod 注解的键名
}

type CRDConfig struct {
	Group     string `json:"group,omitempty"`     // CRD 中的 Group 名称
	Version   string `json:"version"`             // CRD 中的版本
	Resource  string `json:"resource"`            // CRD 中的 resource 名称，一般都是复数形式，比如pods
	Namespace string `json:"namespace,omitempty"` // CRD 中的 namespace 名称
	Name      string `json:"name"`                // CRD 中的名称
	JsonPath  string `json:"jsonPath"`            // JsonPatch中的路径
}

type HTTPMetricConfig struct {
	MetricName string `json:"metricName"` // 指标名称
}

type StorageConfig struct {
	Type          StorageType          `json:"type"`                    // 存储类型
	PodAnnotation *PodAnnotationConfig `json:"podAnnotation,omitempty"` // 适用于 Pod 注解的配置
	CRD           *CRDConfig           `json:"crd,omitempty"`           // 适用于 CRD 的配置
	HTTPMetric    *HTTPMetricConfig    `json:"httpMetric,omitempty"`    // 适用于 HTTP 指标的配
}

type FieldType string

const (
	FieldTypeString FieldType = "string"
	FieldTypeInt    FieldType = "int"
	FieldTypeFloat  FieldType = "float"
)

type JSONPathConfig struct {
	JSONPath  string    `json:"jsonPath"`  // JSONPath 表达式
	FieldType FieldType `json:"fieldType"` // 提取结果的数据类型
}

func (s *StorageConfig) StoreData(factory StorageFactory, data interface{}) error {
	storage, err := factory.GetStorage(s.Type)
	if err != nil {
		return fmt.Errorf("failed to get storage: %w", err)
	}
	switch s.Type {
	case StorageTypePodAnnotation:
		return storage.Store(data, s.PodAnnotation)
	case StorageTypeCRD:
		return storage.Store(data, s.CRD)
	case StorageTypeHTTPMetric:
		return storage.Store(data, s.HTTPMetric)
	default:
		return fmt.Errorf("unsupported storage type: %s", s.Type)
	}
}

func (c *CRDConfig) IsValid() error {
	if c.JsonPath == "" {
		return fmt.Errorf("invalid jsonPath")
	}
	// groupName can be empty
	// if c.GroupName==""{
	// 	return fmt.Errorf("invalid group name")
	// }
	if c.Version == "" {
		return fmt.Errorf("invalid version")
	}
	if c.Resource == "" {
		return fmt.Errorf("invalid resource")
	}
	return nil
}

// ParseTemplate parses the template in CRDConfig
func (c *CRDConfig) ParseTemplate(container *corev1.Container) error {
	v, err := expression.ReplaceValue(c.Namespace, container)
	if err != nil {
		return fmt.Errorf("failed to parse namespace: %w", err)
	}
	c.Namespace = v
	v, err = expression.ReplaceValue(c.Name, container)
	if err != nil {
		return fmt.Errorf("failed to parse name: %w", err)
	}
	c.Name = v
	return nil
}
