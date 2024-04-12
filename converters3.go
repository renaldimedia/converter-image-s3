package main

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/chai2010/webp"
	_ "github.com/go-sql-driver/mysql"
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
	Endpoint      string
	Bucket        string
	SizeAfter     int64
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

	// s3 settings
	s3Endpoint := os.Getenv("S3_ENDPOINT")
	s3AccessKey := os.Getenv("S3_ACCESS_KEY")
	s3SecretKey := os.Getenv("S3_SECRET_KEY")
	s3Bucket := "halorumah"
	s3Folder := "wp-content" // Folder containing images to convert

	// Initialize MariaDB connection
	db, err := sql.Open("mysql", os.Getenv("MYSQL_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Initialize s3 client
	s3Client, err := minio.New(s3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(s3AccessKey, s3SecretKey, ""),
		Secure: true,
	})
	if err != nil {
		log.Fatalf("Error initializing s3 client: %v", err)
	}

	// List objects in s3 folder
	ctx := context.Background()
	s3ObjectsCh := s3Client.ListObjects(ctx, s3Bucket, minio.ListObjectsOptions{
		Prefix:    s3Folder,
		Recursive: true,
	})

	// Initialize a worker pool
	numWorkers := 4
	workerCh := make(chan minio.ObjectInfo, numWorkers)
	var wg sync.WaitGroup
	wg.Add(numWorkers)

	// Start worker goroutines
	for i := 0; i < numWorkers; i++ {
		go convertWorker(ctx, s3Client, s3Bucket, s3Folder, db, workerCh, &wg)
	}

	// Iterate over S3 objects and send them to worker goroutines
	for obj := range s3ObjectsCh {
		if obj.Err != nil {
			log.Printf("Error listing objects: %v", obj.Err)
			continue
		}
		workerCh <- obj
	}

	// Wait for all worker goroutines to finish
	wg.Wait()
	close(workerCh)
}

func convertWorker(ctx context.Context, s3Client *minio.Client, s3Bucket, s3Folder string, db *sql.DB, workerCh <-chan minio.ObjectInfo, wg *sync.WaitGroup) {
	defer wg.Done()

	for obj := range workerCh {
		err := convertAndUpload(ctx, s3Client, s3Bucket, s3Folder, db, obj)
		if err != nil {
			log.Printf("Error processing object '%s': %v", obj.Key, err)
		}
	}
}

func convertAndUpload(ctx context.Context, s3Client *minio.Client, s3Bucket, s3Folder string, db *sql.DB, obj minio.ObjectInfo) error {
	if !isImageFile(obj.Key) {
		return nil
	}

	// Check if the file has been converted before
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM converted_files WHERE filename = ? AND endpoint = ? AND bucket = ? AND (size = ? OR size_after = ?) AND filepath = ?", obj.Key, s3Client.EndpointURL().String(), s3Bucket, obj.Size, obj.Size, filepath.Join(s3Folder, obj.Key)).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to query database: %w", err)
	}
	if count > 0 {
		log.Printf("File '%s' has already been converted. Skipping.", obj.Key)
		return nil
	}

	destPath := filepath.Join("downloaded", obj.Key)

	// Download the image from S3
	err = s3Client.FGetObject(ctx, s3Bucket, obj.Key, destPath, minio.GetObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to download object from S3: %w", err)
	}
	defer os.Remove(destPath)

	log.Printf("Downloaded object from S3: %s", obj.Key)

	// Read the image content
	file, err := os.Open(destPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Decode the image
	img, _, err := image.Decode(file)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// Convert the image to WebP format
	var webpBuf bytes.Buffer
	err = webp.Encode(&webpBuf, img, &webp.Options{Lossless: false, Quality: 65})
	if err != nil {
		return fmt.Errorf("failed to convert image to WebP format: %w", err)
	}

	// Upload the WebP image to S3 with the same filename
	_, err = s3Client.PutObject(ctx, s3Bucket, obj.Key, &webpBuf, int64(webpBuf.Len()), minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("failed to upload WebP image to S3: %w", err)
	}

	// Track the converted file in MariaDB
	_, err = db.ExecContext(ctx, "INSERT INTO converted_files (filename, filepath, size, converted_time, endpoint, bucket, size_after) VALUES (?, ?, ?, ?, ?, ?, ?)", obj.Key, filepath.Join(s3Folder, obj.Key), obj.Size, time.Now(), s3Client.EndpointURL().String(), s3Bucket, int64(webpBuf.Len()))
	if err != nil {
		return fmt.Errorf("failed to insert record into database: %w", err)
	}

	log.Printf("Successfully converted and uploaded %s to WebP format", obj.Key)

	return nil
}

func isImageFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".gif"
}
