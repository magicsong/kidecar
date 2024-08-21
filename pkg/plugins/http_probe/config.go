package httpprobe

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
	Type           StorageType          `json:"type"`                    // 存储类型
	JSONPathConfig *JSONPathConfig      `json:"jsonPathConfig"`          // JSONPath 配置
	PodAnnotation  *PodAnnotationConfig `json:"podAnnotation,omitempty"` // 适用于 Pod 注解的配置
	CRD            *CRDConfig           `json:"crd,omitempty"`           // 适用于 CRD 的配置
	HTTPMetric     *HTTPMetricConfig    `json:"httpMetric,omitempty"`    // 适用于 HTTP 指标的配
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

type EndpointConfig struct {
	URL                string            `json:"url"`                // 目标 URL
	Method             string            `json:"method"`             // HTTP 方法
	Headers            map[string]string `json:"headers"`            // 请求头
	Timeout            int               `json:"timeout"`            // 超时时间（秒）
	ExpectedStatusCode int               `json:"expectedStatusCode"` // 预期的 HTTP 状态码
	StorageConfig      StorageConfig     `json:"storageConfig"`      // 存储配置
}

type HttpProbeConfig struct {
	Endpoints []EndpointConfig `json:"endpoints"` // 多个端点的配置
}
