# S3 File Listing Service

This service provides an API endpoint to list files in an Amazon S3 bucket.

## Getting Started

To get started with this service, follow the steps below.

### Prerequisites

- Go (1.16 or higher)
- Amazon Web Services (AWS) account with S3 access

### Installation

1. Clone the repository:

```bash
git clone https://github.com/orkait/orkait-file-service.git
```

2. Change into the project directory:

```bash
cd orkait-file-service
```

3. Install the dependencies:

```bash
go mod download
```

4. Configure the service:
Create a configuration file named config.json in the project root and populate it with the necessary configuration parameters:

```js
BUCKET_NAME=orkait-file-service
REGION=ap-south-1
DOWNLOAD_URL_TIME_LIMIT=300
PAGINATION_PAGE_SIZE=100
AWS_ACCESS_KEY_ID=your-aws-access-key-id
AWS_SECRET_ACCESS_KEY=your-aws-secret-access-key
```

## Usage

To run the service, execute the following command:

```bash
go run main.go
```
