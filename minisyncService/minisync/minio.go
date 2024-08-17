package minisync

import (
	"context"
	"log"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinioClient wraps the MinIO client and provides additional context for operations on a specific bucket.
// It includes the MinIO client instance and the name of the bucket being operated on.
type MinioClient struct {
	Client     *minio.Client // Client is the MinIO client instance used to interact with MinIO.
	BucketName string        // BucketName is the name of the bucket where operations are performed.
}

// NewMinioClient creates a new MinioClient with the specified endpoint, access key, secret key, and bucket name.
// If the specified bucket does not exist, it attempts to create it. If the bucket already exists, it logs the information.
func NewMinioClient(endpoint, accessKey, secretKey, bucketName string) (*MinioClient, error) {
	log.Printf("Creating MinIO client with endpoint: %s", endpoint)

	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		log.Printf("Error creating MinIO client with endpoint: %s, accessKey: %s", endpoint, accessKey)
		return nil, err
	}

	err = minioClient.MakeBucket(context.Background(), bucketName, minio.MakeBucketOptions{})
	if err != nil {
		exists, errBucketExists := minioClient.BucketExists(context.Background(), bucketName)
		if errBucketExists == nil && exists {
			log.Printf("We already own %s\n", bucketName)
		} else {
			return nil, err
		}
	}

	return &MinioClient{Client: minioClient, BucketName: bucketName}, nil
}

// CreateFile uploads a new file to MinIO, effectively the same as uploading a file.
func (c *MinioClient) CreateFile(relativePath, filePath string) error {
	return c.UploadFile(relativePath, filePath)
}

// UpdateFile updates an existing file in MinIO by re-uploading it.
func (c *MinioClient) UpdateFile(relativePath, filePath string) error {
	return c.UploadFile(relativePath, filePath)
}

// RenameFile renames a file in MinIO by copying it to the new path and deleting the old file.
func (c *MinioClient) RenameFile(oldRelativePath, newRelativePath, newFilePath string) error {
	// Upload the file to the new path
	err := c.UploadFile(newRelativePath, newFilePath)
	if err != nil {
		return err
	}

	// Delete the file from the old path
	err = c.DeleteFile(oldRelativePath)
	if err != nil {
		return err
	}

	return nil
}

// UploadFile uploads a file to the specified bucket in MinIO, preserving the directory structure.
// The relativePath parameter specifies the path within the bucket, and filePath is the local file path to be uploaded.
func (c *MinioClient) UploadFile(relativePath, filePath string) error {
	_, err := c.Client.FPutObject(context.Background(), c.BucketName, relativePath, filePath, minio.PutObjectOptions{})
	return err
}

// DeleteFile deletes a file from the specified bucket in MinIO, preserving the directory structure.
// The relativePath parameter specifies the path within the bucket to the file to be deleted.
func (c *MinioClient) DeleteFile(relativePath string) error {
	err := c.Client.RemoveObject(context.Background(), c.BucketName, relativePath, minio.RemoveObjectOptions{})
	return err
}
