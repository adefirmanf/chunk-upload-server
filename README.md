# Chunk Upload Server

![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)
![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)


A lightweight Go server that implements the FilePond chunked upload protocol for resumable file uploads. This server handles large file uploads by splitting them into chunks, allowing for resume capabilities and better handling of network interruptions.

## Demo
Coming soon

## Features

- ✅ **Chunked Uploads** - Split large files into manageable chunks
- ✅ **Resumable Transfers** - Resume interrupted uploads from where they left off
- ✅ **CORS Support** - Cross-Origin Resource Sharing enabled for web apps
- ✅ **FilePond Compatible** - Works seamlessly with FilePond file upload library
- ✅ **Lightweight** - Simple Go implementation with no external dependencies
- ✅ **Docker Ready** - Easy deployment with Docker

## How It Works

The server implements the FilePond chunked upload protocol:

1. **POST /upload** - Client requests a transfer ID
   - Server creates unique transfer ID (e.g., `a1b2c3d4...`)
   - Server creates temporary directory `tmp/{transferID}/`
   - Server responds with the transfer ID in plain text

2. **PATCH /upload?patch={transferID}** - Client sends file chunks
   - Each request includes headers:
     - `Upload-Offset`: Byte offset of the chunk
     - `Upload-Length`: Total file size
     - `Upload-Name`: File name
   - Server writes chunk at the specified offset
   - Server responds with `Upload-Offset` header containing next expected offset

3. **HEAD /upload?patch={transferID}** - Client checks upload progress (for resume)
   - Server responds with `Upload-Offset` header containing current file size
   - Client can resume upload from this offset

4. **Upload Completion**
   - When all chunks are received, server moves file to final location
   - Temporary directory is cleaned up

## Installation

### Local Development

```bash
# Clone the repository
git clone https://github.com/adefirmanf/chunk-upload-server.git
cd chunk-upload-server

# Run the server
go run main.go
```

The server will start on `http://localhost:8090`

### Docker

```bash
# Build the Docker image
docker build -t chunk-upload-server .

# Run the container
docker run -p 8090:8090 chunk-upload-server

# Run with persistent uploads directory
docker run -p 8090:8090 -v $(pwd)/uploads:/root/tmp chunk-upload-server
```

## API Reference

### POST /upload

Create a new upload transfer.

**Headers:**
- `Upload-Length`: Total file size in bytes (required)
- `Upload-Metadata`: Optional metadata about the upload

**Response:**
- Status: `200 OK`
- Content-Type: `text/plain`
- Body: Transfer ID (e.g., `a1b2c3d4e5f6...`)

**Example:**
```bash
curl -X POST http://localhost:8090/upload \
  -H "Upload-Length: 1000000" \
  -H "Upload-Metadata: filename d29ybGRfZG9taW5hdGlvbl9wbGFuLnBkZg=="
```

### PATCH /upload?patch={transferID}

Upload a file chunk.

**Headers:**
- `Upload-Offset`: Byte offset where this chunk starts (required)
- `Upload-Length`: Total file size in bytes (required)
- `Upload-Name`: File name (optional)

**Body:** Binary chunk data

**Response:**
- Status: `204 No Content`
- Headers:
  - `Upload-Offset`: Next expected byte offset

**Example:**
```bash
curl -X PATCH http://localhost:8090/upload?patch=a1b2c3d4 \
  -H "Upload-Offset: 0" \
  -H "Upload-Length: 1000000" \
  -H "Upload-Name: myfile.pdf" \
  --data-binary @chunk1.bin
```

### HEAD /upload?patch={transferID}

Check current upload progress.

**Response:**
- Status: `200 OK`
- Headers:
  - `Upload-Offset`: Current file size (next expected byte offset)

**Example:**
```bash
curl -I -X HEAD http://localhost:8090/upload?patch=a1b2c3d4
```

## Usage with FilePond

This server is designed to work with [FilePond](https://pondjs.com/), a JavaScript file upload library.

```javascript
import FilePond from 'filepond';

const pond = FilePond.create({
  server: {
    url: 'http://localhost:8090',
    process: {
      url: '/upload',
      method: 'POST',
      withCredentials: false,
      headers: {},
      timeout: 7000,
    },
    patch: {
      url: '/upload?patch=',
      method: 'PATCH',
      withCredentials: false,
      headers: {},
      timeout: 7000,
    },
    revert: null,
    restore: null,
    load: null,
    fetch: null,
  },
  chunkUploads: true,
  chunkSize: 5000000, // 5MB chunks
});
```

## Configuration

The server uses the following defaults:

- **Port:** `8090`
- **Upload Directory:** `./tmp`
- **CORS:** Enabled for all origins (`*`)

To modify these settings, edit the constants in `main.go`:

```go
const uploadDir = "./tmp"  // Change upload directory
```

For the port, modify the `main()` function:

```go
http.ListenAndServe(":8090", nil)  // Change port
```

## File Storage

- Uploaded files are temporarily stored in `tmp/{transferID}/data`
- On successful completion, files are moved to `tmp/{filename}`
- You can modify the final storage location in the `handlePatch()` function

## Security Considerations

⚠️ **This is a basic implementation for development purposes.** For production use, consider:

- Adding authentication/authorization
- Validating file types and sizes
- Implementing rate limiting
- Securing file storage paths
- Restricting CORS origins
- Adding virus scanning
- Implementing file cleanup for abandoned uploads

## License

MIT

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Author

[adefirmanf](https://github.com/adefirmanf)
