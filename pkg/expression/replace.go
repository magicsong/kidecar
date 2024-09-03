package expression

import (
	"fmt"
	"os"
	"regexp"

	corev1 "k8s.io/api/core/v1"
)

// 表达式格式：
// ${SELF:VAR_NAME}：表示sidecar自身的环境变量。
// ${POD:VAR_NAME}：表示 Pod 的环境变量。

const (
	pattern = `\$\{(SELF|POD):([^}]+)\}`
)

func ReplaceValue(value string, container *corev1.Container) (string, error) {
	// 检查是否符合表达式格式
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(value)
	if matches == nil {
		return value, nil
	}

	envType := matches[1]
	envName := matches[2]

	var envValue string
	var found bool
	if envType == "SELF" {
		envValue, found = os.LookupEnv(envName)
	} else if envType == "POD" {
		// 从容器的环境变量中查找
		for _, envVar := range container.Env {
			if envVar.Name == envName {
				envValue = envVar.Value
				found = true
				break
			}
		}
	} else {
		return "", fmt.Errorf("unknown environment variable type: %s", envType)
	}

	if !found {
		return "", fmt.Errorf("environment variable %s not found", envName)
	}

	return envValue, nil
}
