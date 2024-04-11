# Image Conversion Utility

This Go script is a utility for converting images to the WebP format and uploading them to a cloud storage service (Wasabi in this case) while keeping track of the conversions in a MariaDB database. It provides a solution for efficiently converting and managing image files.

## Features

- Converts images to WebP format for improved web performance.
- Utilizes concurrent processing to optimize conversion speed.
- Integrates with cloud storage (Wasabi) for seamless image uploading.
- Tracks conversion history using a MariaDB database.

## Prerequisites

Before running the script, ensure you have the following installed and configured:

- Go programming language (at least version 1.21.X)
- MariaDB database
- MinIO client for interacting with Wasabi (if using a different cloud storage provider, adjust accordingly)
- Access to a Wasabi account with API credentials

## Setup

1. Clone the repository:

```bash
git clone https://github.com/yourusername/your-repo.git
cd your-repo
```

2. Install dependencies:

```bash
go mod tidy
```

3. Set up environment variables:

   Create a `.env` file with the following variables:

   ```dotenv
   MYSQL_URL=mysql://username:password@localhost:3306/database
   S3_ENDPOINT=your-wasabi-endpoint
   S3_ACCESS_KEY=your-wasabi-access-key
   S3_SECRET_KEY=your-wasabi-secret-key
   ```

4. Create a MariaDB database:

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
go run main.go
```

## License

This project is licensed under the [MIT License](LICENSE).

## Contributions

Contributions are welcome! Feel free to submit issues or pull requests.

---

Feel free to customize the README further based on your preferences and additional information about your project.
