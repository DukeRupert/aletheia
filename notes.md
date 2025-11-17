
## Storage

### AWS implementation
```go
// Load AWS config
cfg, err := config.LoadDefaultConfig(context.Background())
if err != nil {
    log.Fatal(err)
}

// Create S3 client
s3Client := s3.NewFromConfig(cfg)

// Use S3 storage instead
fileStorage := storage.NewS3Storage(
    s3Client,
    "my-bucket-name",
    "us-east-1",
    "https://cdn.myapp.com", // CloudFront URL
)
```
```
```
```
