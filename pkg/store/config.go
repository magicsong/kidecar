package store

import "fmt"

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
	CRDName   string `json:"crdName"`   // CRD 名称
	FieldName string `json:"fieldName"` // CRD 中的字段名
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
