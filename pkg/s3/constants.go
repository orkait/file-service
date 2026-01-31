package s3

const (
	// MaxBatchSize is the maximum number of files allowed per batch operation
	MaxBatchSize = 100

	// DefaultMaxWorkers is the default number of concurrent workers for batch operations
	DefaultMaxWorkers = 10

	ErrorUploadCancelled   = "upload cancelled"
	ErrorDownloadCancelled = "download cancelled"
	ErrorMaxBatchExceeded  = "maximum %d files allowed per batch"
	ErrorNoFilesProvided   = "no files provided. Use 'files' as the form-data field name"
	ErrorNoPathsProvided   = "no paths provided"
)
