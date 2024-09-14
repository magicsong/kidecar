package hot_update

import (
	"fmt"
	"regexp"
)

const (
	// pluginName is the name of the plugin.
	pluginName = "hot_update"

	// URLKey is the key of the URL in the request body.
	URLKey = "url"

	// LoadPatchTypeSignal is the type of load patch that sends a signal to the main container.
	LoadPatchTypeSignal = "signal"
	// LoadPatchTypeRequest is the type of load patch that sends a request to the main container.
	LoadPatchTypeRequest = "request"
)

func validateConfig(hotUpdateConfig *HotUpdateConfig) error {
	if hotUpdateConfig.LoadPatchType != LoadPatchTypeRequest && hotUpdateConfig.LoadPatchType != LoadPatchTypeSignal {
		return fmt.Errorf("loadPatchType is empty")
	}
	if hotUpdateConfig.LoadPatchType == LoadPatchTypeSignal {
		if hotUpdateConfig.Signal.SignalName == "" {
			return fmt.Errorf("SignalName is empty")
		}
	}
	if hotUpdateConfig.LoadPatchType == LoadPatchTypeRequest {
		if hotUpdateConfig.Request.Address == "" {
			return fmt.Errorf("address is empty")
		}
		if hotUpdateConfig.Request.Port == 0 {
			return fmt.Errorf("port is empty")
		}
	}

	if hotUpdateConfig.FileDir == "" {
		return fmt.Errorf("fileDir is empty")
	}

	return nil
}

func isValidVersion(version string) bool {
	re := regexp.MustCompile(`^v\d+(?:\.\d+)*$`)
	return re.MatchString(version)
}
