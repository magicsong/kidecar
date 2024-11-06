package store

import (
	"context"
	"fmt"

	"github.com/magicsong/kidecar/pkg/constants"
	"github.com/magicsong/kidecar/pkg/info"
	"gopkg.in/yaml.v3"
)

type PersistentConfig struct {
	Type   string            `json:"type"`
	Result map[string]string `json:"result"`
}

func (p *PersistentConfig) GetPersistenceInfo() error {
	if p == nil {
		return fmt.Errorf("persistent config is nil")
	}
	if p.Type == "" {
		return fmt.Errorf("persistent config type is invalid")
	}

	cm, err := info.GetConfigmap(context.TODO(), constants.SidecarResultConfigMapName, constants.SidecarResultConfigMapNamespace)
	if err != nil {
		return fmt.Errorf("failed to get hotupdate configmap %s/%s: %v", constants.SidecarResultConfigMapName, constants.SidecarResultConfigMapNamespace, err)
	}

	nameAndNamespace, err := info.GetCurrentPodInfo()
	if err != nil {
		return fmt.Errorf("failed to get current pod namespace and name: %w", err)
	}

	p.Result = map[string]string{}

	persistentInfo := map[string]map[string]string{}
	if info, ok := cm.Data[nameAndNamespace]; ok {
		err := yaml.Unmarshal([]byte(info), &persistentInfo)
		if err != nil {
			return fmt.Errorf("failed to unmarshal hotUpdateInfo: %v", err)
		}

		if res, ok := persistentInfo[p.Type]; ok {
			p.Result = res
		}
	} else {
		for _, v := range cm.Data {
			err := yaml.Unmarshal([]byte(v), &persistentInfo)
			if err != nil {
				return fmt.Errorf("failed to unmarshal hotUpdateInfo: %v", err)
			}
			if res, ok := persistentInfo[p.Type]; ok {
				for version, url := range res {
					p.Result[version] = url
				}
			}
		}
	}

	return nil
}

func (p *PersistentConfig) SetPersistenceInfo() error {
	if p == nil || p.Type == "" {
		return fmt.Errorf("persistent config is invalid")
	}

	cm, err := info.GetConfigmap(context.TODO(), constants.SidecarResultConfigMapName, constants.SidecarResultConfigMapNamespace)
	if err != nil {
		return fmt.Errorf("failed to get hotupdate configmap %s/%s: %v", constants.SidecarResultConfigMapName, constants.SidecarResultConfigMapNamespace, err)
	}
	nameAndNamespace, err := info.GetCurrentPodInfo()
	if err != nil {
		return fmt.Errorf("failed to get current pod namespace and name: %w", err)
	}

	persistentInfo := map[string]map[string]string{}

	if len(cm.Data) == 0 {
		cm.Data = make(map[string]string)
	}

	if info, ok := cm.Data[nameAndNamespace]; ok {
		err := yaml.Unmarshal([]byte(info), &persistentInfo)
		if err != nil {
			return fmt.Errorf("failed to unmarshal hotUpdateInfo: %v", err)
		}
	}

	if _, ok := persistentInfo[p.Type]; !ok {
		persistentInfo[p.Type] = make(map[string]string)
	}
	for k, v := range p.Result {
		persistentInfo[p.Type][k] = v
	}

	persistentInfoBytes, err := yaml.Marshal(persistentInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal hotUpdateInfo: %v", err)
	}

	cm.Data[nameAndNamespace] = string(persistentInfoBytes)

	if _, err := info.UpdateConfigmap(context.TODO(), cm); err != nil {
		return fmt.Errorf("failed to update hotupdate configmap %s/%s: %v", constants.SidecarResultConfigMapName, constants.SidecarResultConfigMapNamespace, err)
	}

	return nil

}
