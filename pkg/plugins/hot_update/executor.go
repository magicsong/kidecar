package hot_update

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/magicsong/kidecar/pkg/store"
	"golang.org/x/mod/semver"

	"github.com/magicsong/kidecar/pkg/template"
)

func (h *hotUpdate) hotUpdateHandle(w http.ResponseWriter, r *http.Request) {

	err := h.downloadHotUpdateFile(w, r)
	if err != nil {
		h.log.Error(err, "Failed to download file")
		return
	}

	h.log.Info("File downloaded and saved successfully")

	// 根据输入的Config，判断采用那种方式触发更新
	switch h.config.LoadPatchType {
	case LoadPatchTypeSignal:
		err := h.loadHotUpdateFileBySignal()
		if err != nil {
			h.log.Error(err, "Failed to load hot update file by signal")
			h.result.Result = fmt.Sprintf("%s: Failed to load hot update file by signal: %s", h.result.Version, err)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "File downloaded and update successfully")
		h.result.Result = fmt.Sprintf("%s: Update success", h.result.Version)
	case LoadPatchTypeRequest:
		err := h.loadHotUpdateFileByRequest()
		if err != nil {
			h.log.Error(err, "Failed to load hot update file by request")
			return
		}
	}

	err = h.storeData()
	if err != nil {
		h.log.Error(err, "Failed to store data")
		return
	}

	err = h.storeDataToConfigmap()
	if err != nil {
		h.log.Error(err, "failed to store data to configmap")
		return
	}
}

func (h *hotUpdate) downloadHotUpdateFile(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "only POST requests are allowed")
		return fmt.Errorf("only POST requests are allowed")
	}

	// get version from request
	version := r.FormValue("version")
	if version == "" || !isValidVersion(version) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "please provide a valid version")
		return fmt.Errorf("please provide a valid version")
	}
	h.result.Version = version

	// get file url from request
	url := r.FormValue(URLKey)
	if url == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "please provide a valid URL")
		return fmt.Errorf("please provide a valid URL")
	}
	h.result.Url = url

	err := h.downloadFileByUrl()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "failed to download file: %v", err)
		return err
	}

	return nil
}

func (h *hotUpdate) downloadFileByUrl() error {
	// Check if the hot update file  exists, if not, create it
	if _, err := os.Stat(h.config.FileDir); os.IsNotExist(err) {
		err := os.MkdirAll(h.config.FileDir, 0755)
		if err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}
	}

	// get file name from url
	fileURL, err := http.NewRequest(http.MethodGet, h.result.Url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request for URL: %w", err)
	}
	filePath := filepath.Base(fileURL.URL.Path)

	// full save path
	savePath := filepath.Join(h.config.FileDir, filePath)

	// check if file exists, if exists, delete it
	if _, err := os.Stat(savePath); err == nil {
		err := os.Remove(savePath)
		if err != nil {
			return fmt.Errorf("failed to delete file: %w", err)
		}
	}

	// download file
	resp, err := http.Get(h.result.Url)
	if err != nil {
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	// save file
	out, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}

func (h *hotUpdate) loadHotUpdateFileBySignal() error {
	// get process list
	psout, err := exec.Command("ps", "aux").Output()
	if err != nil {
		return fmt.Errorf("failed to get process list, err: %v", err)
	}

	processName := h.config.Signal.ProcessName
	signal := h.config.Signal.SignalName
	processes := strings.Split(string(psout), "\n")
	for _, process := range processes {
		if strings.Contains(process, processName) {
			processInfos := strings.Fields(process)
			var pid string
			for _, processInfo := range processInfos {
				// 判断 processInfo是否为数字
				if _, err := strconv.Atoi(processInfo); err != nil {
					continue
				}
				pid = processInfo
				cmd := exec.Command("kill", "-s", signal, pid)
				err := cmd.Run()
				if err != nil {
					return fmt.Errorf("failed to send signal to PID, signal: %v , pid: %v, processInfo: %v, err: %v", signal, pid, processInfos, err)
				}
				h.log.Info("Signal sent successfully", "signal", signal, "pid", pid, " processInfo: ", processInfos)
				return nil
			}
		}
	}
	return fmt.Errorf("process not found, processName: %v", processName)
}

func (h *hotUpdate) loadHotUpdateFileByRequest() error {
	return nil
}

func (h *hotUpdate) storeData() error {

	h.log.Info("store update result, ", "result: ", h.result.Result)
	err := template.ParseConfig(h.config.StorageConfig)
	if err != nil {
		return fmt.Errorf("failed to parse config: %v", err)
	}

	return h.config.StorageConfig.StoreData(h.StorageFactory, h.result.Result)
}

func (h *hotUpdate) storeDataToConfigmap() error {

	persistentResult := &store.PersistentConfig{
		Type: pluginName,
		Result: map[string]string{
			h.result.Version: h.result.Url,
		},
	}

	err := persistentResult.SetPersistenceInfo()
	if err != nil {
		return fmt.Errorf("failed to set hot update result to configmap")
	}

	return nil
}

func (h *hotUpdate) setHotUpdateConfigWhenStart() error {

	persistentResult := &store.PersistentConfig{
		Type: pluginName,
	}
	err := persistentResult.GetPersistenceInfo()
	if err != nil {
		return fmt.Errorf("failed to GetPersistenceInfo of %v ", pluginName)
	}

	// 未重启
	if len(persistentResult.Result) == 0 {
		h.log.Info("sidecar result has not the pod info")
		return nil
	}

	version := ""
	url := ""
	for v, u := range persistentResult.Result {
		if semver.Compare(version, v) < 0 {
			version = v
			url = u
		}
	}

	h.result.Version = version
	h.result.Url = url

	// down load
	err = h.downloadFileByUrl()
	if err != nil {
		return fmt.Errorf("failed to download file: %v", err)
	}

	switch h.config.LoadPatchType {
	case LoadPatchTypeSignal:
		err := h.loadHotUpdateFileBySignal()
		if err != nil {
			h.log.Error(err, "Failed to load hot update file by signal")
			h.result.Result = fmt.Sprintf("%s: Failed to load hot update file by signal: %s", h.result.Version, err)
			return fmt.Errorf("failed to load hot update file by signal")
		}
		h.result.Result = fmt.Sprintf("%s: Update success", h.result.Version)
	case LoadPatchTypeRequest:
		err := h.loadHotUpdateFileByRequest()
		if err != nil {
			h.log.Error(err, "Failed to load hot update file by request")
			return err
		}
	}

	err = h.storeData()
	if err != nil {
		return fmt.Errorf("failed to store data: %v", err)
	}
	return nil
}
