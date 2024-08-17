package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/mwiater/minisync/minisyncService/minisync"
	"golang.org/x/sys/windows/registry"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
	"golang.org/x/sys/windows/svc/mgr"
)

// Constants related to the service configuration.
const (
	serviceName        = "MiniSync"                                       // The internal name of the Windows service.
	serviceDisplayName = "MiniSync Service"                               // The display name of the service.
	serviceDescription = "A service to sync files from Windows to MinIO." // The description of the service.
)

// myService represents the Windows service and its behavior.
type myService struct{}

// Execute is the main entry point for the service execution. It handles various control requests
// like start, stop, pause, continue, and shutdown. It also manages the service's state and logs
// important events.
func (m *myService) Execute(args []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (bool, uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	s <- svc.Status{State: svc.StartPending}
	s <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}

	elog, err := eventlog.Open(serviceName)
	if err != nil {
		return false, 1
	}
	defer elog.Close()

	elog.Info(1, "Starting: minisyncService")
	go minisyncService(elog)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				s <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				elog.Info(1, serviceName+" stopping")
				s <- svc.Status{State: svc.StopPending}
				return false, 0
			case svc.Pause:
				s <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
				ticker.Stop()
			case svc.Continue:
				s <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
				ticker = time.NewTicker(2 * time.Second)
			default:
				elog.Warning(1, "unexpected control request")
			}
		case <-ticker.C:
			elog.Info(1, "Tick")
		}
	}
}

// compareFiles checks if the local file and the remote file are identical by comparing their sizes and modification times.
// Returns true if the files are identical, otherwise false.
func compareFiles(localPath string, remoteObject *minio.ObjectInfo) bool {
	localFileInfo, err := os.Stat(localPath)
	if err != nil {
		log.Printf("Failed to stat local file %s: %v", localPath, err)
		return false
	}

	if localFileInfo.Size() == remoteObject.Size && localFileInfo.ModTime().Equal(remoteObject.LastModified) {
		return true
	}

	return false
}

// minisyncService initializes the Minisync service by loading environment variables, setting up logging,
// and starting directory monitoring and file synchronization tasks with MinIO.
func minisyncService(elog *eventlog.Log) {
	MINISYNC_BACKUPFOLDER, _ := fetchEnvironmentVariable("MINISYNC_BACKUPFOLDER")
	MINISYNC_LOGFOLDER, _ := fetchEnvironmentVariable("MINISYNC_LOGFOLDER")
	MINISYNC_MINIO_ENDPOINT, _ := fetchEnvironmentVariable("MINISYNC_MINIO_ENDPOINT")
	MINISYNC_MINIO_BUCKETNAME, _ := fetchEnvironmentVariable("MINISYNC_MINIO_BUCKETNAME")
	MINISYNC_MINIO_BACKUPFREQUENCYSECONDS, _ := fetchEnvironmentVariable("MINISYNC_MINIO_BACKUPFREQUENCYSECONDS")
	MINISYNC_MINIO_ACCESS_KEY, _ := fetchEnvironmentVariable("MINISYNC_MINIO_ACCESS_KEY")
	MINISYNC_MINIO_SECRET_KEY, _ := fetchEnvironmentVariable("MINISYNC_MINIO_SECRET_KEY")

	elog.Info(1, "Set: logFile")

	logFile, err := os.OpenFile(MINISYNC_LOGFOLDER+"\\MiniSync.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		elog.Info(1, "Failed to open log file")
		log.Fatalf("Failed to open log file: %v", err)
	}

	log.SetOutput(logFile)
	log.Println("Starting MiniSync service...")
	log.Printf("Connecting to MinIO server at %s with access key %s", MINISYNC_MINIO_ENDPOINT, MINISYNC_MINIO_ACCESS_KEY)
	log.Println("Set: minioClient")

	elog.Info(1, "Set: minioClient")
	minioClient, err := minisync.NewMinioClient(MINISYNC_MINIO_ENDPOINT, MINISYNC_MINIO_ACCESS_KEY, MINISYNC_MINIO_SECRET_KEY, MINISYNC_MINIO_BUCKETNAME)
	if err != nil {
		elog.Info(1, "Failed to create Minio client")
		log.Fatalf("Failed to create Minio client: %v", err)
	}

	go minisync.MonitorDirectory(MINISYNC_BACKUPFOLDER, minioClient)

	backupFrequencySeconds, _ := strconv.Atoi(MINISYNC_MINIO_BACKUPFREQUENCYSECONDS)
	ticker := time.NewTicker(time.Duration(backupFrequencySeconds) * time.Second)
	for range ticker.C {
		elog.Info(1, "Starting full sync cycle")
		err := filepath.Walk(MINISYNC_BACKUPFOLDER, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			relativePath, err := filepath.Rel(MINISYNC_BACKUPFOLDER, path)
			if err != nil {
				log.Printf("Failed to get relative path for %s: %v", path, err)
				return err
			}

			if info.IsDir() {
				log.Printf("Found directory: %s", relativePath)
				return nil
			}

			// Check if the file exists on MinIO
			remoteObject, err := minioClient.Client.StatObject(context.Background(), minioClient.BucketName, relativePath, minio.StatObjectOptions{})
			if err != nil {
				if minio.ToErrorResponse(err).Code == "NoSuchKey" {
					// File does not exist on remote, upload it
					log.Printf("Uploading new file %s to MinIO", relativePath)
					err = minioClient.CreateFile(relativePath, path)
					if err != nil {
						log.Printf("Failed to upload file %s: %v", path, err)
						return err
					}
				} else {
					// Some other error occurred
					log.Printf("Failed to stat remote file %s: %v", relativePath, err)
					return err
				}
			} else {
				// File exists on remote, compare it with the local file
				if compareFiles(path, &remoteObject) {
					// Files are identical, do nothing
					log.Printf("File %s is identical on local and remote, skipping", relativePath)
				} else {
					// Files are not identical, update the remote file
					log.Printf("Updating file %s on MinIO", relativePath)
					err = minioClient.UpdateFile(relativePath, path)
					if err != nil {
						log.Printf("Failed to update file %s: %v", path, err)
						return err
					}
				}
			}

			return nil
		})

		if err != nil {
			log.Printf("Failed to walk directory: %v", err)
			elog.Info(1, "Failed to walk directory")
		} else {
			log.Println("Full sync cycle completed")
			elog.Info(1, "Full sync cycle completed")
		}

		// Check for files on the remote that don't exist locally and delete them
		doneCh := make(chan struct{})
		defer close(doneCh)

		objectCh := minioClient.Client.ListObjects(context.Background(), minioClient.BucketName, minio.ListObjectsOptions{
			Prefix:    "", // Change prefix if you want to limit the scope
			Recursive: true,
		})

		for object := range objectCh {
			if object.Err != nil {
				log.Printf("Error listing objects: %v", object.Err)
				continue
			}

			localPath := filepath.Join(MINISYNC_BACKUPFOLDER, object.Key)
			if _, err := os.Stat(localPath); os.IsNotExist(err) {
				// File exists on remote but not locally, delete it
				log.Printf("Deleting remote file %s that does not exist locally", object.Key)
				err = minioClient.DeleteFile(object.Key)
				if err != nil {
					log.Printf("Failed to delete remote file %s: %v", object.Key, err)
				}
			}
		}
	}
}

// runService runs the Minisync service, either in debug mode or as a standard Windows service.
func runService(name string, isDebug bool) {
	var err error
	if isDebug {
		err = debug.Run(name, &myService{})
	} else {
		err = svc.Run(name, &myService{})
	}
	if err != nil {
		log.Fatalf("%s service failed: %v", name, err)
	}
}

// installService installs the Minisync service on the local machine. It sets up the service configuration
// and registers it with the Windows Service Manager.
func installService(name, exepath string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err == nil {
		s.Close()
		return fmt.Errorf("service %s already exists", name)
	}

	s, err = m.CreateService(name, exepath, mgr.Config{
		DisplayName: serviceDisplayName,
		StartType:   mgr.StartAutomatic,
		Description: serviceDescription,
	}, "is", "auto-started")
	if err != nil {
		return err
	}
	defer s.Close()

	err = eventlog.InstallAsEventCreate(name, eventlog.Info|eventlog.Warning|eventlog.Error)
	if err != nil {
		s.Delete()
		return fmt.Errorf("setupEventLogSource() failed: %s", err)
	}
	return nil
}

// uninstallService removes the Minisync service from the local machine, including its registration
// with the Windows Service Manager and associated event logs.
func uninstallService(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("service %s is not installed", name)
	}
	defer s.Close()

	err = s.Delete()
	if err != nil {
		return err
	}

	err = eventlog.Remove(name)
	if err != nil {
		return fmt.Errorf("RemoveEventLogSource() failed: %s", err)
	}
	return nil
}

// startService starts the Minisync service using the Windows Service Manager.
func startService(name string) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("could not access service: %v", err)
	}
	defer s.Close()

	err = s.Start("is", "manual-started")
	if err != nil {
		return fmt.Errorf("could not start service: %v", err)
	}
	return nil
}

// controlService sends a control command to the Minisync service, such as stop, pause, or continue.
// It waits until the service reaches the desired state or a timeout occurs.
func controlService(name string, c svc.Cmd, to svc.State) error {
	m, err := mgr.Connect()
	if err != nil {
		return err
	}
	defer m.Disconnect()

	s, err := m.OpenService(name)
	if err != nil {
		return fmt.Errorf("could not access service: %v", err)
	}
	defer s.Close()

	status, err := s.Control(c)
	if err != nil {
		return fmt.Errorf("could not send control=%d: %v", c, err)
	}

	timeout := time.Now().Add(10 * time.Second)
	for status.State != to {
		if timeout.Before(time.Now()) {
			return fmt.Errorf("timeout waiting for service to go to state=%d", to)
		}
		time.Sleep(300 * time.Millisecond)
		status, err = s.Query()
		if err != nil {
			return fmt.Errorf("could not retrieve service status: %v", err)
		}
	}
	return nil
}

// stopService stops the Minisync service using the Windows Service Manager.
func stopService(name string) error {
	return controlService(name, svc.Stop, svc.Stopped)
}

// pauseService pauses the Minisync service using the Windows Service Manager.
func pauseService(name string) error {
	return controlService(name, svc.Pause, svc.Paused)
}

// continueService resumes the Minisync service after it has been paused.
func continueService(name string) error {
	return controlService(name, svc.Continue, svc.Running)
}

// usage displays the command-line usage information for the Minisync service management commands.
func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <command>\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  install   Install the service\n")
	fmt.Fprintf(os.Stderr, "  uninstall Uninstall the service\n")
	fmt.Fprintf(os.Stderr, "  start     Start the service\n")
	fmt.Fprintf(os.Stderr, "  stop      Stop the service\n")
	fmt.Fprintf(os.Stderr, "  pause     Pause the service\n")
	fmt.Fprintf(os.Stderr, "  continue  Resume the service\n")
	os.Exit(2)
}

// main is the entry point for the Minisync service application. It determines whether the application
// is running as a Windows service or in interactive mode and executes the appropriate commands.
func main() {
	isWindowsService, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("failed to determine if we are running in an interactive session: %v", err)
	}
	if isWindowsService {
		runService(serviceName, false)
		return
	}

	if len(os.Args) < 2 {
		usage()
		return
	}

	cmd := os.Args[1]
	exepath, err := os.Executable()
	if err != nil {
		log.Fatalf("failed to get executable path: %v", err)
	}

	switch cmd {
	case "install":
		err = installService(serviceName, exepath)
	case "uninstall":
		err = uninstallService(serviceName)
	case "start":
		err = startService(serviceName)
	case "stop":
		err = stopService(serviceName)
	case "pause":
		err = pauseService(serviceName)
	case "continue":
		err = continueService(serviceName)
	default:
		usage()
	}
	if err != nil {
		log.Fatalf("failed to %s %s: %v", cmd, serviceName, err)
	}
}

// fetchEnvironmentVariable retrieves the value of an environment variable from the Windows registry.
// It opens the registry key, fetches the variable value, and returns it. If the variable is not found,
// an error is returned.
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
