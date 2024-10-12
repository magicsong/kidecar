package store

import (
	"context"
	"testing"

	"github.com/magicsong/kidecar/pkg/constants"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/magicsong/kidecar/pkg/info"
	corev1 "k8s.io/api/core/v1"
)

func TestPersistentConfig_GetPersistenceInfo(t *testing.T) {

	gomonkey.ApplyFunc(info.GetConfigmap, func(ctx context.Context, name string, namespace string) (*corev1.ConfigMap, error) {
		return &corev1.ConfigMap{
			Data: map[string]string{
				"pod1-default": `    hot_update:
      v1: url1
      v2: url2
    probe:
      test: test1`,
			},
		}, nil
	})

	gomonkey.ApplyFunc(info.GetCurrentPodInfo, func() (string, error) {
		return "pod1-default", nil
	})
	type fields struct {
		Type   string
		Result map[string]string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "test1",
			fields: fields{
				Type: constants.SidecarResultType,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PersistentConfig{
				Type:   tt.fields.Type,
				Result: tt.fields.Result,
			}
			if err := p.GetPersistenceInfo(); (err != nil) != tt.wantErr {
				t.Errorf("GetPersistenceInfo() error = %v, wantErr %v", err, tt.wantErr)
			}
			if p.Result["v1"] != "url1" {
				t.Errorf("GetPersistenceInfo() p.Result: %v", p.Result)
			}
		})
	}
}

func TestPersistentConfig_SetPersistenceInfo(t *testing.T) {
	gomonkey.ApplyFunc(info.GetConfigmap, func(ctx context.Context, name string, namespace string) (*corev1.ConfigMap, error) {
		return &corev1.ConfigMap{}, nil
	})

	gomonkey.ApplyFunc(info.UpdateConfigmap, func(ctx context.Context, cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
		return nil, nil
	})

	gomonkey.ApplyFunc(info.GetCurrentPodInfo, func() (string, error) {
		return "pod1-default", nil
	})

	type fields struct {
		Type   string
		Result map[string]string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "test1",
			fields: fields{
				Type: constants.SidecarResultType,
				Result: map[string]string{
					"v2": "url2",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PersistentConfig{
				Type:   tt.fields.Type,
				Result: tt.fields.Result,
			}
			if err := p.SetPersistenceInfo(); (err != nil) != tt.wantErr {
				t.Errorf("SetPersistenceInfo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
