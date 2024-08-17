package minisync

import (
	"context"
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	"github.com/minio/minio-go/v7"
)

// MonitorDirectory monitors the specified directory for changes and synchronizes those changes
// to MinIO. It watches for changes in both the directory and its subdirectories, responding to
// events such as file creation, modification, deletion, and renaming.
func MonitorDirectory(sourceFolder string, minioClient *MinioClient) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	err = filepath.Walk(sourceFolder, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			err = watcher.Add(path)
			if err != nil {
				log.Fatal(err)
			}
		}
		return nil
	})
	if err != nil {
		log.Fatal(err)
	}

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			handleEvent(sourceFolder, event, watcher, minioClient)
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("Error:", err)
		}
	}
}

// handleEvent processes file system events and performs the appropriate MinIO operations
// based on the type of event. It handles file creation, modification, deletion, and renaming
// while preserving the directory structure in MinIO.
func handleEvent(sourceFolder string, event fsnotify.Event, watcher *fsnotify.Watcher, minioClient *MinioClient) {
	relativePath, err := filepath.Rel(sourceFolder, event.Name)
	if err != nil {
		log.Printf("Failed to get relative path for %s: %v", event.Name, err)
		return
	}

	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		log.Println("Created file:", event.Name)
		if !isDir(event.Name) {
			err := minioClient.CreateFile(relativePath, event.Name)
			if err != nil {
				log.Printf("Failed to upload file: %v", err)
			}
		} else {
			err = watcher.Add(event.Name)
			if err != nil {
				log.Printf("Failed to watch new directory: %v", err)
			}
		}
	case event.Op&fsnotify.Write == fsnotify.Write:
		log.Println("Modified file:", event.Name)
		if !isDir(event.Name) {
			err := minioClient.UpdateFile(relativePath, event.Name)
			if err != nil {
				log.Printf("Failed to upload file: %v", err)
			}
		}
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		log.Println("Deleted file or directory:", event.Name)
		if !isDir(event.Name) {
			err := minioClient.DeleteFile(relativePath)
			if err != nil {
				log.Printf("Failed to delete file: %v", err)
			}
		} else {
			// Handle directory deletion by deleting all files under that directory in MinIO
			err := minioClient.DeleteDirectory(relativePath)
			if err != nil {
				log.Printf("Failed to delete directory %s in MinIO: %v", relativePath, err)
			}
		}
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		log.Println("Renamed file or directory:", event.Name)
		if !isDir(event.Name) {
			err := minioClient.DeleteFile(relativePath)
			if err != nil {
				log.Printf("Failed to delete old file after rename: %v", err)
			}
			// Note: In this simplified version, we assume the new file will be handled separately by a Create event.
		} else {
			// Handle directory rename by deleting the old directory and expecting the new one to be created
			err := minioClient.DeleteDirectory(relativePath)
			if err != nil {
				log.Printf("Failed to delete old directory after rename: %v", err)
			}
			// Note: The new directory should trigger a Create event
		}
	}
}

// DeleteDirectory deletes all files in the specified directory from the MinIO bucket.
func (c *MinioClient) DeleteDirectory(relativePath string) error {
	doneCh := make(chan struct{})
	defer close(doneCh)

	objectCh := c.Client.ListObjects(context.Background(), c.BucketName, minio.ListObjectsOptions{
		Prefix:    relativePath + "/", // Ensure we're only deleting within this directory
		Recursive: true,
	})

	for object := range objectCh {
		if object.Err != nil {
			log.Printf("Error listing objects in directory %s: %v", relativePath, object.Err)
			continue
		}

		err := c.DeleteFile(object.Key)
		if err != nil {
			log.Printf("Failed to delete file %s: %v", object.Key, err)
		}
	}

	return nil
}

// isDir checks if the specified path is a directory. It returns true if the path
// is a directory and false otherwise. If an error occurs while retrieving the
// file information, it returns false.
func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
