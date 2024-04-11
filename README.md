# Image Conversion Utility

This Go script is designed for optimizing image files as part of storage cleanup for our company. To ensure no files are lost, the script converts images to the WebP format, preserving their quality while reducing file size.

## Workflow

1. **List Files:** The script scans the storage to identify image files.

2. **Check Conversion Status:** It verifies whether each image has been previously converted by querying a MariaDB database.

3. **Conversion and Upload:**
   - For unconverted images, the script downloads the file, converts it to WebP format, and uploads the optimized version back to its original path.
   - Concurrent processing is employed to enhance conversion efficiency.

4. **Cleanup:**
   - After successful conversion and upload, the local copy of the original image is deleted to save storage space.

## Prerequisites

Before running the script, ensure the following are set up:
- Go programming language (version 1.21.X +)
- MariaDB database
- MinIO client for interaction with Wasabi (adjust accordingly if using a different cloud storage provider)
- Access to a Wasabi account with API credentials

## Setup

1. **Clone the Repository:**

2. **Install Dependencies:**
```bash
go mod tidy
```

3. **Set Environment Variables:**
   - Create a `.env` file with the required variables:
   ```dotenv
   MYSQL_URL=mysql://username:password@localhost:3306/database
   S3_ENDPOINT=your-wasabi-endpoint
   S3_ACCESS_KEY=your-wasabi-access-key
   S3_SECRET_KEY=your-wasabi-secret-key
   ```

4. **Database Setup:**
   - Create a MariaDB database:
   ```sql
   CREATE DATABASE your_database_name;
   USE your_database_name;

   CREATE TABLE converted_files (
       id INT AUTO_INCREMENT PRIMARY KEY,
       filename VARCHAR(255),
       filepath VARCHAR(255),
       size BIGINT,
       converted_time DATETIME
   );
   ```

## Usage

Run the script using the following command:
```bash
go run converters3.go
```
