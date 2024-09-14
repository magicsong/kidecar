package hot_update

//func Test_hotUpdate_setHotUpdateConfigWhenStart(t *testing.T) {
//
//	gomonkey.ApplyFunc(info.GetCurrentPodInfo, func() (string, error) {
//		return "pod1-default", nil
//	})
//	p := &store.PersistentConfig{
//		Result: map[string]string{
//			"v1.0":   "url1",
//			"v3":     "url3",
//			"v0.0.5": "url5",
//			"v6.0.1": "url6",
//			"v2.0":   "url2",
//		},
//	}
//	gomonkey.ApplyMethod(p, "GetPersistenceInfo", func() error {
//		p.Result = map[string]string{
//			"v1.0": "url1",
//			"v2.0": "url2",
//		}
//		return nil
//	})
//
//	type fields struct {
//		config         HotUpdateConfig
//		StorageFactory store.StorageFactory
//		status         *HotUpdateStatus
//		result         *HotUpdateResult
//		log            logr.Logger
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		wantErr bool
//	}{
//		{},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			h := &hotUpdate{
//				config:         tt.fields.config,
//				StorageFactory: tt.fields.StorageFactory,
//				status:         tt.fields.status,
//				result:         tt.fields.result,
//				log:            tt.fields.log,
//			}
//			if err := h.setHotUpdateConfigWhenStart(); (err != nil) != tt.wantErr {
//				t.Errorf("setHotUpdateConfigWhenStart() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}
