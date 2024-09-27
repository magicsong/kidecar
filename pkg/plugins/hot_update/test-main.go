package hot_update

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func updateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		fmt.Fprintf(w, "Only POST requests are allowed")
		return
	}

	url := r.FormValue("url")
	if url == "" {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Please provide a valid URL")
		return
	}

	// 指定保存文件的目录为 /app/downloads
	dir := "/app/downloads"
	// 如果目录不存在，创建它
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "Failed to create directory: %v", err)
			return
		}
	}

	// 从 URL 中获取文件名
	fileURL, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to create request for URL: %v", err)
		return
	}
	filePath := filepath.Base(fileURL.URL.Path)

	// 完整的保存路径
	savePath := filepath.Join(dir, filePath)

	// 检查文件是否已存在，如果存在则删除
	if _, err := os.Stat(savePath); err == nil {
		os.Remove(savePath)
	}

	// 下载文件
	resp, err := http.Get(url)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to download file: %v", err)
		return
	}
	defer resp.Body.Close()

	// 创建文件并保存内容
	out, err := os.Create(savePath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to create file: %v", err)
		return
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to save file: %v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "File downloaded and saved successfully")

	// 执行 ps 命令获取进程列表
	psout, err := exec.Command("ps", "aux").Output()
	if err != nil {
		fmt.Println("Error getting process list:", err)
		return
	}

	// 从环境变量获取主进程名字
	processName := os.Getenv("PROCESS_NAME")
	if processName == "" {
		fmt.Println("Please set the PROCESS_NAME environment variable")
		return
	}

	// 从环境变量获取信号量，如果未设置，默认使用 Signhub
	signal := os.Getenv("SIGNAL")
	if signal == "" {
		signal = "SIGHUP"
	}

	lines := strings.Split(string(psout), "\n")
	for _, line := range lines {
		if strings.Contains(line, processName) {
			fields := strings.Fields(line)
			pid := fields[1]
			cmd := exec.Command("kill", "-s", signal, pid)
			err := cmd.Run()
			if err != nil {
				fmt.Printf("Error sending signal %s to PID %s: %v\n", signal, pid, err)
			} else {
				fmt.Printf("Sent signal %s to PID %s successfully\n", signal, pid)
			}
		}
	}

}

func main() {

	http.HandleFunc("/update", updateHandler)

	// 启动服务，监听 5000 端口
	err := http.ListenAndServe(":5000", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
		return
	}
}
