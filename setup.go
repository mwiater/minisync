package main

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/mwiater/minisync/minisyncService/minisync"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sys/windows/registry"
)

// minisyncService contains the embedded binary for the Minisync service.
// This binary is saved to the file system and executed as needed.
//
//go:embed bin/MiniSyncService.exe
var minisyncService embed.FS

// Config represents the structure of the form data used to configure the Minisync service.
type Config struct {
	BackupFolder           string `json:"backupFolder"`
	LogFolder              string `json:"logFolder"`
	MinioEndpoint          string `json:"minioEndpoint"`
	MinioKey               string `json:"minioKey"`
	MinioSecret            string `json:"minioSecret"`
	MinioBucketName        string `json:"miniobucketName"`
	BackupFrequencySeconds string `json:"backupFrequencySeconds"`
}

// App represents the main application struct.
type App struct {
	ctx context.Context
}

// NewApp creates a new instance of the App struct.
func NewApp() *App {
	return &App{}
}

// startup is called when the application starts up. It sets the application context.
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// domReady is called after the front-end resources have been loaded.
func (a App) domReady(ctx context.Context) {
	// Additional actions can be added here if needed
}

// beforeClose is called when the application is about to quit. Returning true prevents the application from closing.
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	return false
}

// shutdown is called when the application is terminating. It performs any necessary cleanup.
func (a *App) shutdown(ctx context.Context) {
	// Add cleanup code here if needed
}

// BrowseFolder opens a dialog allowing the user to select a folder. The selected folder path is returned.
func (a *App) BrowseFolder() (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Select Folder",
	})
}

// GetServiceStatus retrieves the current status of the Minisync service.
func (a *App) GetServiceStatus() (string, error) {
	serviceStatus, err := minisync.GetServiceStatus("MiniSync")
	if err != nil {
		log.Fatalf("Error getting MiniSync status: %v", err)
		return "", err
	}

	return serviceStatus, nil
}

// ServiceControl manages the Minisync service by executing commands such as start, stop, install, and uninstall.
func (a *App) ServiceControl(command string) string {
	exePath, err := os.Executable()
	if err != nil {
		return "ERROR!"
	}
	dir := filepath.Dir(exePath)

	exePath = filepath.Join(dir, "MiniSyncService.exe")

	result := runCommand(exePath, command)

	return result
}

// runCommand executes the specified command on the given file and passes the necessary environment variables.
func runCommand(fileName string, command string) string {
	var cmd *exec.Cmd
	if command == "install" || command == "start" {
		key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`, registry.QUERY_VALUE)
		if err != nil {
			log.Fatalf("Failed to open registry key: %v", err)
		}
		defer key.Close()

		MINISYNC_BACKUPFOLDER, _ := fetchEnvironmentVariable("MINISYNC_BACKUPFOLDER")
		MINISYNC_LOGFOLDER, _ := fetchEnvironmentVariable("MINISYNC_LOGFOLDER")
		MINISYNC_MINIO_ENDPOINT, _ := fetchEnvironmentVariable("MINISYNC_MINIO_ENDPOINT")
		MINISYNC_MINIO_BUCKETNAME, _ := fetchEnvironmentVariable("MINISYNC_MINIO_BUCKETNAME")
		MINISYNC_MINIO_BACKUPFREQUENCYSECONDS, _ := fetchEnvironmentVariable("MINISYNC_MINIO_BACKUPFREQUENCYSECONDS")
		MINISYNC_MINIO_ACCESS_KEY, _ := fetchEnvironmentVariable("MINISYNC_MINIO_ACCESS_KEY")
		MINISYNC_MINIO_SECRET_KEY, _ := fetchEnvironmentVariable("MINISYNC_MINIO_SECRET_KEY")

		if MINISYNC_BACKUPFOLDER == "" || MINISYNC_LOGFOLDER == "" || MINISYNC_MINIO_ENDPOINT == "" || MINISYNC_MINIO_BUCKETNAME == "" || MINISYNC_MINIO_BACKUPFREQUENCYSECONDS == "" || MINISYNC_MINIO_ACCESS_KEY == "" || MINISYNC_MINIO_SECRET_KEY == "" {
			log.Fatal("One or more environment variables are not set")
		}

		env := append(os.Environ(),
			"MINISYNC_BACKFOLDER="+MINISYNC_BACKUPFOLDER,
			"MINISYNC_LOGFOLDER="+MINISYNC_LOGFOLDER,
			"MINISYNC_MINIO_ENDPOINT="+MINISYNC_MINIO_ENDPOINT,
			"MINISYNC_MINIO_BUCKETNAME="+MINISYNC_MINIO_BUCKETNAME,
			"MINISYNC_MINIO_BACKUPFREQUENCYSECONDS="+MINISYNC_MINIO_BACKUPFREQUENCYSECONDS,
			"MINISYNC_MINIO_ACCESS_KEY="+MINISYNC_MINIO_ACCESS_KEY,
			"MINISYNC_MINIO_SECRET_KEY="+MINISYNC_MINIO_SECRET_KEY,
		)
		cmd = exec.Command(fileName, command)
		cmd.Env = env
	} else {
		cmd = exec.Command(fileName, command)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error executing %s: %v\n", command, err)
	}

	log.Printf("Output of %s:\n%s\n", command, string(output))

	if string(output) == "" {
		result := "Success: " + command
		return result
	}

	return string(output)
}

// SubmitForm handles the form submission from the frontend, updating environment variables and managing the Minisync service.
func (a *App) SubmitForm(config Config) (string, error) {

	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`, registry.QUERY_VALUE)
	if err != nil {
		log.Fatalf("Failed to open registry key: %v", err)
	}
	defer key.Close()

	exists, err := keyExists(`SYSTEM\CurrentControlSet\Control\Session Manager\Environment`)
	if err != nil {
		log.Fatalf("Error checking if key exists: %v", err)
	}
	if !exists {
		log.Println("Key does not exist, creating it.")
	} else {
		log.Println("Key exists.")
	}

	envVars := map[string]string{
		"MINISYNC_BACKFOLDER":                   config.BackupFolder,
		"MINISYNC_LOGFOLDER":                    config.LogFolder,
		"MINISYNC_MINIO_ENDPOINT":               config.MinioEndpoint,
		"MINISYNC_MINIO_BUCKETNAME":             config.MinioBucketName,
		"MINISYNC_MINIO_BACKUPFREQUENCYSECONDS": config.BackupFrequencySeconds,
		"MINISYNC_MINIO_ACCESS_KEY":             config.MinioKey,
		"MINISYNC_MINIO_SECRET_KEY":             config.MinioSecret,
	}

	for key, value := range envVars {
		err := setEnvironmentVariable(key, value)
		if err != nil {
			log.Printf("Error setting key %s: %v", key, err)
		}
	}

	// TO DO: NEED TO FIX / REMOVE
	if 1 == 2 {
		const HWND_BROADCAST = 0xFFFF
		const WM_SETTINGCHANGE = 0x001A
		const SMTO_ABORTIFHUNG = 0x0002

		user32 := syscall.NewLazyDLL("user32.dll")
		sendMessageTimeout := user32.NewProc("SendMessageTimeoutW")

		envPtr, err := syscall.UTF16PtrFromString("Environment")
		if err != nil {
			log.Fatalf("Failed to convert string to UTF16 pointer: %v", err)
		}

		result, _, err := sendMessageTimeout.Call(
			uintptr(HWND_BROADCAST),
			uintptr(WM_SETTINGCHANGE),
			uintptr(0),
			uintptr(unsafe.Pointer(envPtr)),
			uintptr(SMTO_ABORTIFHUNG),
			uintptr(5000),
		)

		if result == 0 {
			log.Fatalf("Failed to broadcast environment change: %v", err)
		}
	}

	log.Println("System environment variables updated successfully.")

	exePath, err := saveMinisyncService("MiniSyncService.exe")
	if err != nil {
		log.Fatalf("Failed to save child binary: %v", err)
	}

	runCommand(exePath, "stop")
	runCommand(exePath, "uninstall")
	runCommand(exePath, "install")
	runCommand(exePath, "start")

	serviceStatus, err := a.GetServiceStatus()
	if err != nil {
		log.Fatalf("Error unmarshalling .env file: %v", err)
	}

	return serviceStatus, nil
}

// UninstallMinisyncService uninstalls the Minisync service and removes related environment variables.
func UninstallMinisyncService() {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`, registry.SET_VALUE)
	if err != nil {
		log.Fatalf("Failed to open registry key: %v", err)
	}
	defer key.Close()

	log.Println("Unsetting environment variables:")
	unsetEnvironmentVariable("MINISYNC_BACKUPFOLDER")
	unsetEnvironmentVariable("MINISYNC_LOGFOLDER")
	unsetEnvironmentVariable("MINISYNC_MINIO_ENDPOINT")
	unsetEnvironmentVariable("MINISYNC_MINIO_BUCKETNAME")
	unsetEnvironmentVariable("MINISYNC_MINIO_BACKUPFREQUENCYSECONDS")
	unsetEnvironmentVariable("MINISYNC_MINIO_ACCESS_KEY")
	unsetEnvironmentVariable("MINISYNC_MINIO_SECRET_KEY")
}

// saveMinisyncService writes the embedded Minisync service executable to a file.
// It returns the full path to the saved executable or an error if the operation fails.
func saveMinisyncService(fileName string) (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	dir := filepath.Dir(exePath)

	exePath = filepath.Join(dir, fileName)

	binaryData, err := minisyncService.ReadFile("bin/MiniSyncService.exe")
	if err != nil {
		return "", fmt.Errorf("failed to read embedded binary file: %w", err)
	}

	err = os.WriteFile(exePath, binaryData, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to write the binary file: %w", err)
	}

	return exePath, nil
}

// setEnvironmentVariable sets the specified environment variable in the Windows registry.
func setEnvironmentVariable(name, value string) error {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`, registry.SET_VALUE)
	if err != nil {
		log.Fatalf("Failed to open registry key: %v", err)
	}
	defer key.Close()

	err = key.SetStringValue(name, value)
	if err != nil {
		return fmt.Errorf("failed to set environment variable %s: %w", name, err)
	}
	return nil
}

// fetchEnvironmentVariable retrieves the specified environment variable from the Windows registry.
func fetchEnvironmentVariable(name string) (string, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`, registry.QUERY_VALUE)
	if err != nil {
		log.Fatalf("Failed to open registry key: %v", err)
	}
	defer key.Close()

	value, _, err := key.GetStringValue(name)
	if err != nil {
		return "", fmt.Errorf("failed to fetch environment variable %s: %w", name, err)
	}
	return value, nil
}

// unsetEnvironmentVariable removes the specified environment variable from the Windows registry.
func unsetEnvironmentVariable(name string) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Session Manager\Environment`, registry.SET_VALUE)
	if err != nil {
		log.Fatalf("Failed to open registry key: %v", err)
	}
	defer key.Close()

	err = key.DeleteValue(name)
	if err != nil {
		log.Printf("Failed to unset environment variable %s: %v", name, err)
	} else {
		log.Printf("Unset environment variable %s successfully", name)
	}
}

// keyExists checks if the specified registry key exists.
func keyExists(path string) (bool, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, path, registry.QUERY_VALUE)
	if err != nil {
		if err == registry.ErrNotExist {
			return false, nil
		}
		return false, fmt.Errorf("failed to open registry key: %w", err)
	}
	key.Close()
	return true, nil
}
