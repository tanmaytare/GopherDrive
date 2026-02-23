# GopherDrive

A distributed file management system built with Go, featuring both gRPC and REST API interfaces for file metadata management and processing.

## Overview

GopherDrive is a high-performance file metadata service that provides:
- **gRPC Service** for efficient file metadata management
- **REST API** for file uploads and operations
- **Worker Pool** for parallel file processing
- **MySQL Backend** for persistent metadata storage
- **SHA256 Hashing** for file integrity verification

## Architecture

```
┌─────────────┐
│  REST API   │ (Port 8080)
│ :8080/files │
└──────┬──────┘
       │
       ├──────┐
       │      └─────────────────────┐
       ▼                            ▼
┌─────────────────┐        ┌──────────────────┐
│  Worker Pool    │        │  gRPC Service   │ (Port 50051)
│  (Processes:    │◄───────┤  :50051         │
│   - SHA256      │        │  MetadataService│
│   - File Ops)   │        └──────────────────┘
└────────┬────────┘               ▲
         │                        │
         └────────────┬───────────┘
                      ▼
              ┌──────────────────┐
              │     MySQL DB     │
              │  (gopherdrive)   │
              │   (Metadata)     │
              └──────────────────┘
```

## Features

- **File Registration**: Register and track files with unique IDs
- **Metadata Management**: Store file path, size, hash, and status
- **Concurrent Processing**: Process multiple files simultaneously using worker pool
- **Status Tracking**: Monitor file processing status (PENDING, PROCESSING, COMPLETED, FAILED)
- **gRPC & REST APIs**: Choose between high-performance gRPC or standard HTTP REST

## Prerequisites

- **Go**: 1.25 or higher
- **MySQL**: 5.7 or higher
- **Protocol Buffers**: `protoc` 3.0+
- **Go Protobuf Plugins**:
  - `protoc-gen-go`
  - `protoc-gen-go-grpc`

### macOS Installation

```bash
# Install Go (using Homebrew)
brew install go

# Install protoc
brew install protobuf

# Install Go protobuf plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Install MySQL
brew install mysql
```

### Linux Installation (Ubuntu/Debian)

```bash
# Install Go
wget https://go.dev/dl/go1.25.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.25.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Install protoc
sudo apt-get install -y protobuf-compiler

# Install MySQL
sudo apt-get install -y mysql-server

# Install Go protobuf plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

## Installation

1. **Clone the repository:**
   ```bash
   git clone https://github.com/tanmaytare/gopherdrive.git
   cd gopherdrive
   ```

2. **Install dependencies:**
   ```bash
   go mod download
   ```

3. **Set up MySQL Database:**
   
   Start MySQL service and run the following SQL commands:
   
   ```sql
   CREATE DATABASE gopherdrive;
   
   USE gopherdrive;
   
   CREATE TABLE metadata (
       id VARCHAR(36) PRIMARY KEY,
       file_path VARCHAR(255),
       sha256 VARCHAR(64),
       file_size BIGINT,
       status VARCHAR(20),
       created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
   );
   ```

4. **Build the project (optional):**
   ```bash
   go build ./cmd/grpc
   go build ./cmd/rest
   ```

## Configuration

### Database Connection

The database connection string is configured in:
- **gRPC Server**: [cmd/grpc/main.go](cmd/grpc/main.go#L47)
- **Default**: `root:rootuser@tcp(127.0.0.1:3306)/gopherdrive`

Modify the connection string if you have different MySQL credentials:
```go
db, err := sql.Open("mysql", "your_username:your_password@tcp(127.0.0.1:3306)/gopherdrive")
```

### Ports

- **gRPC Server**: `50051`
- **REST API**: `8080`
- **Worker Pool**: `5` concurrent workers

## Usage

### Running the Project

You need to run two services:

**Terminal 1 - Start gRPC Server:**
```bash
cd gopherdrive
go run cmd/grpc/main.go
```

Expected output:
```
gRPC running on :50051
```

**Terminal 2 - Start REST API:**
```bash
cd gopherdrive
go run cmd/rest/main.go
```

Expected output:
```
REST running on :8080
```

### API Examples

#### Health Check

```bash
curl http://localhost:8080/healthz
```

Response:
```
OK
```

#### Upload File

```bash
curl -X POST -F "file=@/path/to/your/file.txt" http://localhost:8080/files
```

Response:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000"
}
```

#### Upload Multiple Files in Parallel

```bash
for i in {1..5}; do 
  curl -X POST -F "file=@test.txt" http://localhost:8080/files &
done
```

### gRPC Client Example

```go
package main

import (
	"context"
	pb "github.com/tanmaytare/gopherdrive/proto"
	"google.golang.org/grpc"
)

func main() {
	conn, _ := grpc.Dial("localhost:50051", grpc.WithInsecure())
	defer conn.Close()
	
	client := pb.NewMetadataServiceClient(conn)
	
	// Register a file
	resp, _ := client.RegisterFile(context.Background(), &pb.RegisterRequest{
		Id:   "file-123",
		Path: "./data/file-123",
		Size: 1024,
	})
	
	// Get file metadata
	file, _ := client.GetFile(context.Background(), &pb.GetRequest{
		Id: "file-123",
	})
	
	// Update file status
	client.UpdateStatus(context.Background(), &pb.UpdateRequest{
		Id:     "file-123",
		Hash:   "abc123...",
		Status: "COMPLETED",
	})
}
```

## Project Structure

```
gopherdrive/
├── cmd/
│   ├── grpc/
│   │   └── main.go           # gRPC server implementation
│   └── rest/
│       └── main.go           # REST API server implementation
├── internal/
│   ├── repository/
│   │   └── mysql.go          # MySQL database operations
│   ├── service/              # Business logic (optional)
│   └── worker/
│       └── worker.go         # Worker pool for concurrent processing
├── proto/
│   ├── metadata.proto        # Protocol Buffer definition
│   ├── metadata.pb.go        # Generated protobuf code
│   └── metadata_grpc.pb.go   # Generated gRPC service code
├── data/                     # Uploaded files storage directory
├── go.mod                    # Go module definition
├── go.sum                    # Go dependencies
└── README.md                 # This file
```

## API Documentation

### gRPC Services

#### MetadataService

**RegisterFile**
- Register a new file in the system
- Input: `RegisterRequest` (id, path, size)
- Output: `RegisterResponse`
- Status: `PENDING` by default

**UpdateStatus**
- Update file processing status and hash
- Input: `UpdateRequest` (id, hash, status)
- Output: `UpdateResponse`
- Status values: `PENDING`, `PROCESSING`, `COMPLETED`, `FAILED`

**GetFile**
- Retrieve file metadata by ID
- Input: `GetRequest` (id)
- Output: `GetResponse` (id, path, hash, size, status)

### REST Endpoints

| Endpoint | Method | Description | Example |
|----------|--------|-------------|---------|
| `/healthz` | GET | Health check | `curl http://localhost:8080/healthz` |
| `/files` | POST | Upload file | `curl -F "file=@test.txt" http://localhost:8080/files` |

## Database Schema

### metadata Table

| Column | Type | Description |
|--------|------|-------------|
| id | VARCHAR(36) | Unique file identifier (UUID) |
| file_path | VARCHAR(255) | Path to the uploaded file |
| sha256 | VARCHAR(64) | SHA256 hash of the file |
| file_size | BIGINT | File size in bytes |
| status | VARCHAR(20) | Current processing status |
| created_at | TIMESTAMP | Record creation timestamp |

## Troubleshooting

### gRPC Server Won't Start

**Error**: `connection refused` or `cannot assign requested address`

**Solution**:
```bash
# Verify MySQL is running
mysql -u root -p

# Check if port 50051 is already in use
lsof -i :50051

# Kill the process if needed
kill -9 <PID>
```

### REST API Connection Error

**Error**: `failed to connect to gRPC server`

**Solution**:
```bash
# Ensure gRPC server is running in another terminal
# Verify gRPC is listening on port 50051
netstat -an | grep 50051
```

### MySQL Connection Error

**Error**: `connection refused` or `access denied for user`

**Solution**:
```bash
# Verify MySQL credentials in cmd/grpc/main.go
# Check MySQL connection string format: username:password@tcp(host:port)/database
# Test MySQL connection
mysql -u root -prootuser -h 127.0.0.1
```

### Proto Files Not Generated

**Error**: `undefined` errors for proto types

**Solution**:
```bash
# Regenerate proto files
cd gopherdrive
export PATH="$GOPATH/bin:$PATH"
protoc --go_out=paths=source_relative:. --go-grpc_out=paths=source_relative:. proto/metadata.proto
```

## Performance Considerations

- **Worker Pool**: Default 5 concurrent workers for processing
- **File Size**: No hard limit, but consider database storage
- **Concurrent Uploads**: Limited by system resources and worker pool size
- **gRPC vs REST**: gRPC provides ~3-5x better performance for metadata operations

## Future Enhancements

- [ ] Distributed file storage (S3, HDFS)
- [ ] Advanced authentication (OAuth2, JWT)
- [ ] Metrics and monitoring (Prometheus)
- [ ] Request rate limiting
- [ ] File deduplication
- [ ] Async processing with message queues
- [ ] Web UI dashboard
- [ ] Comprehensive error handling and logging

## Dependencies

- `google.golang.org/grpc` - gRPC framework
- `google.golang.org/protobuf` - Protocol Buffers
- `github.com/go-sql-driver/mysql` - MySQL driver
- `github.com/google/uuid` - UUID generation

## Development

### Running Tests

```bash
go test ./...
```

### Code Formatting

```bash
go fmt ./...
```

### Build for Production

```bash
# Build gRPC server
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o grpc-server ./cmd/grpc

# Build REST server
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o rest-server ./cmd/rest
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Author

Our Team :
1. Tanmay Tare
2. Pranav Ratnalikar
3. Vishal Kumar
4. Amey Mahulkar
5. Shivam Padalkar

## Acknowledgments

- Go community for excellent standard library
- Protocol Buffers for efficient serialization
- gRPC for high-performance RPC framework
- MySQL for reliable data storage

## Support

For issues, questions, or suggestions, please open an [issue](https://github.com/tanmaytare/gopherdrive/issues) on GitHub.

## Changelog

### Version 1.0.0
- Initial release
- gRPC metadata service
- REST API for file uploads
- MySQL backend
- Worker pool for concurrent processing
