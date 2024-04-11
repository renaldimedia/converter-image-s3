package main

import (
	"bytes"
	"context"
	"database/sql"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chai2010/webp"
	_ "github.com/go-sql-driver/mysql" // Importing MySQL driver

	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// TrackRecord represents the tracked information of a converted file
type TrackRecord struct {
	Filename      string
	Filepath      string
	Size          int64
	ConvertedTime time.Time
}

func main() {
	logFile, err := os.OpenFile("conversion.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}
	defer logFile.Close()

	log.SetOutput(logFile)

	err = godotenv.Load("local.env")
	if err != nil {
		log.Fatalf("Some error occurred. Err: %s", err)
	}

	// Wasabi settings
	wasabiEndpoint := os.Getenv("S3_ENDPOINT")
	wasabiAccessKey := os.Getenv("S3_ACCESS_KEY")
	wasabiSecretKey := os.Getenv("S3_SECRET_KEY")
	wasabiBucket := "halorumah"
	wasabiFolder := "wp-content" // Folder containing images to convert

	// Initialize MariaDB connection
	db, err := sql.Open("mysql", os.Getenv("MYSQL_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize Wasabi client
	wasabiClient, err := minio.New(wasabiEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(wasabiAccessKey, wasabiSecretKey, ""),
		Secure: true,
	})
	if err != nil {
		log.Fatalf("Error initializing Wasabi client: %v", err)
	}

	// List objects in Wasabi folder
	ctx := context.Background()
	wasabiObjects := wasabiClient.ListObjects(ctx, wasabiBucket, minio.ListObjectsOptions{
		Prefix:    wasabiFolder,
		Recursive: true,
	})

	destFolder := "./downloaded"

	// Create a buffered channel to control the number of concurrent conversions
	concurrencyLimit := 2
	concurrencySem := make(chan struct{}, concurrencyLimit)

	// WaitGroup to wait for all conversions to finish
	var wg sync.WaitGroup

	for obj := range wasabiObjects {
		if obj.Err != nil {
			log.Printf("Error listing objects: %v", obj.Err)
			continue
		}

		// Acquire a semaphore to limit concurrency
		concurrencySem <- struct{}{}

		wg.Add(1)
		go func(obj minio.ObjectInfo) {
			defer wg.Done()
			defer func() { <-concurrencySem }()

			// Check if the object is an image file
			if strings.HasSuffix(obj.Key, ".jpg") || strings.HasSuffix(obj.Key, ".jpeg") || strings.HasSuffix(obj.Key, ".png") || strings.HasSuffix(obj.Key, ".gif") {
				// Check if the file has been converted before
				var count int
				err := db.QueryRow("SELECT COUNT(*) FROM converted_files WHERE filename = ?", obj.Key).Scan(&count)
				if err != nil {
					log.Fatal(err)
				}
				if count > 0 {
					log.Printf("File '%s' has already been converted. Skipping.", obj.Key)
					return
				}
				destPath := filepath.Join(destFolder, obj.Key)
				// Download the image from Wasabi
				err = wasabiClient.FGetObject(ctx, wasabiBucket, obj.Key, destPath, minio.GetObjectOptions{})
				if err != nil {
					log.Printf("Error downloading object from Wasabi: %v", err)
					return
				}
				log.Printf("Downloaded object from Wasabi: %s", obj.Key)

				file, err := os.Open(destPath)
				if err != nil {
					log.Printf("Error opening downloaded file '%s': %v", destPath, err)
					return
				}
				defer file.Close()

				// Read the image content
				imageBytes, err := io.ReadAll(file)
				if err != nil {
					log.Printf("Error reading image content: %v", err)
					return
				}

				// Decode the image
				img, _, err := image.Decode(bytes.NewReader(imageBytes))
				if err != nil {
					log.Printf("Error decoding image: %v", err)
					return
				}

				// Convert the image to WebP format
				webpBytes := new(bytes.Buffer)
				err = webp.Encode(webpBytes, img, &webp.Options{Lossless: false})
				if err != nil {
					log.Printf("Error converting image to WebP format: %v", err)
					return
				}

				// Upload the WebP image to Wasabi with the same filename
				_, err = wasabiClient.PutObject(ctx, wasabiBucket, obj.Key, bytes.NewReader(webpBytes.Bytes()), int64(webpBytes.Len()), minio.PutObjectOptions{})
				if err != nil {
					log.Printf("Error uploading WebP image to Wasabi: %v", err)
					return
				}

				// Track the converted file in MariaDB
				_, err = db.Exec("INSERT INTO converted_files (filename, filepath, size, converted_time) VALUES (?, ?, ?, ?)", obj.Key, filepath.Join(wasabiFolder, obj.Key), obj.Size, time.Now())
				if err != nil {
					log.Fatal(err)
				}

				log.Printf("Successfully converted and uploaded %s to WebP format", obj.Key)

				err = os.Remove(destPath) // Delete the downloaded file
				if err != nil {
					log.Printf("Error deleting file: %v", err)
					return
				}
				log.Printf("File deleted successfully: %s", destPath)
			}
		}(obj)
	}

	// Wait for all conversions to finish
	wg.Wait()
}
