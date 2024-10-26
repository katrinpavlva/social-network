package datab

import (
	"cloud.google.com/go/storage"
	"context"
	"io"
	"log"
	"net/url"
)

func StoreToCloud(ctx context.Context, client *storage.Client, bucketName, objectName string, file io.Reader) (string, error) {
	wc := client.Bucket(bucketName).Object(objectName).NewWriter(ctx)
	if _, err := io.Copy(wc, file); err != nil {
		return "", err
	}
	if err := wc.Close(); err != nil {
		return "", err
	}

	log.Println("Uploaded to cloud")

	// Construct the URL for embedding
	publicURL := "https://storage.googleapis.com/" + bucketName + "/" + url.PathEscape(objectName)
	return publicURL, nil
}
