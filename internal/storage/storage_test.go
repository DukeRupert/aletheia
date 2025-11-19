package storage

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Helper function to create a test image
func createTestImage(width, height int, format string) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Fill with a simple pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8((x * 255) / width),
				G: uint8((y * 255) / height),
				B: 128,
				A: 255,
			})
		}
	}

	var buf bytes.Buffer
	switch format {
	case "jpeg", "jpg":
		err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
		if err != nil {
			return nil, err
		}
	case "png":
		err := png.Encode(&buf, img)
		if err != nil {
			return nil, err
		}
	default:
		err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})
		if err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// Helper function to create a multipart file header from bytes
func createFileHeader(filename string, data []byte) (*multipart.FileHeader, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(part, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	// Parse the multipart form to extract the file header
	reader := multipart.NewReader(body, writer.Boundary())
	form, err := reader.ReadForm(int64(len(data)) + 1024)
	if err != nil {
		return nil, err
	}

	if len(form.File["file"]) == 0 {
		return nil, io.EOF
	}

	return form.File["file"][0], nil
}

func TestNewLocalStorage(t *testing.T) {
	tests := []struct {
		name      string
		basePath  string
		baseURL   string
		wantErr   bool
		cleanupFn func()
	}{
		{
			name:     "creates directory if not exists",
			basePath: "./testdata/new-storage",
			baseURL:  "http://localhost:8080/uploads",
			wantErr:  false,
			cleanupFn: func() {
				os.RemoveAll("./testdata")
			},
		},
		{
			name:     "uses existing directory",
			basePath: "./testdata/existing",
			baseURL:  "http://localhost:8080/uploads",
			wantErr:  false,
			cleanupFn: func() {
				os.RemoveAll("./testdata")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.cleanupFn != nil {
				defer tt.cleanupFn()
			}

			// Pre-create directory for "existing" test
			if tt.name == "uses existing directory" {
				err := os.MkdirAll(tt.basePath, 0755)
				if err != nil {
					t.Fatalf("failed to create test directory: %v", err)
				}
			}

			storage, err := NewLocalStorage(tt.basePath, tt.baseURL)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewLocalStorage() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if storage == nil {
					t.Error("NewLocalStorage() returned nil storage")
					return
				}

				// Verify directory was created
				if _, err := os.Stat(tt.basePath); os.IsNotExist(err) {
					t.Errorf("Directory %s was not created", tt.basePath)
				}
			}
		})
	}
}

func TestLocalStorage_Save(t *testing.T) {
	// Create temporary test directory
	testDir := "./testdata/save-test"
	defer os.RemoveAll("./testdata")

	storage, err := NewLocalStorage(testDir, "http://localhost:8080/uploads")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	tests := []struct {
		name     string
		filename string
		format   string
		width    int
		height   int
		wantErr  bool
	}{
		{
			name:     "save jpeg image",
			filename: "test.jpg",
			format:   "jpeg",
			width:    100,
			height:   100,
			wantErr:  false,
		},
		{
			name:     "save png image",
			filename: "test.png",
			format:   "png",
			width:    200,
			height:   150,
			wantErr:  false,
		},
		{
			name:     "save with spaces in name",
			filename: "test image.jpg",
			format:   "jpeg",
			width:    100,
			height:   100,
			wantErr:  false,
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test image
			imageData, err := createTestImage(tt.width, tt.height, tt.format)
			if err != nil {
				t.Fatalf("Failed to create test image: %v", err)
			}

			// Create file header
			fileHeader, err := createFileHeader(tt.filename, imageData)
			if err != nil {
				t.Fatalf("Failed to create file header: %v", err)
			}

			// Save the file
			filename, err := storage.Save(ctx, fileHeader)

			if (err != nil) != tt.wantErr {
				t.Errorf("Save() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify filename is not empty
				if filename == "" {
					t.Error("Save() returned empty filename")
					return
				}

				// Verify file exists on disk
				fullPath := filepath.Join(testDir, filename)
				if _, err := os.Stat(fullPath); os.IsNotExist(err) {
					t.Errorf("File %s was not created", fullPath)
					return
				}

				// Verify file extension matches
				ext := filepath.Ext(filename)
				expectedExt := filepath.Ext(tt.filename)
				if ext != expectedExt {
					t.Errorf("File extension = %s, want %s", ext, expectedExt)
				}

				// Verify file content can be read
				savedData, err := os.ReadFile(fullPath)
				if err != nil {
					t.Errorf("Failed to read saved file: %v", err)
					return
				}

				if len(savedData) == 0 {
					t.Error("Saved file is empty")
				}
			}
		})
	}
}

func TestLocalStorage_Delete(t *testing.T) {
	testDir := "./testdata/delete-test"
	defer os.RemoveAll("./testdata")

	storage, err := NewLocalStorage(testDir, "http://localhost:8080/uploads")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	ctx := context.Background()

	// First, save a test file
	imageData, err := createTestImage(100, 100, "jpeg")
	if err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	fileHeader, err := createFileHeader("test.jpg", imageData)
	if err != nil {
		t.Fatalf("Failed to create file header: %v", err)
	}

	filename, err := storage.Save(ctx, fileHeader)
	if err != nil {
		t.Fatalf("Failed to save test file: %v", err)
	}

	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{
			name:     "delete existing file",
			filename: filename,
			wantErr:  false,
		},
		{
			name:     "delete non-existent file",
			filename: "nonexistent.jpg",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := storage.Delete(ctx, tt.filename)

			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify file no longer exists
				fullPath := filepath.Join(testDir, tt.filename)
				if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
					t.Errorf("File %s still exists after deletion", fullPath)
				}
			}
		})
	}
}

func TestLocalStorage_GetURL(t *testing.T) {
	baseURL := "http://localhost:8080/uploads"
	storage, err := NewLocalStorage("./testdata/url-test", baseURL)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer os.RemoveAll("./testdata")

	tests := []struct {
		name     string
		filename string
		want     string
	}{
		{
			name:     "simple filename",
			filename: "test.jpg",
			want:     "http://localhost:8080/uploads/test.jpg",
		},
		{
			name:     "filename with timestamp",
			filename: "1234567890_uuid.jpg",
			want:     "http://localhost:8080/uploads/1234567890_uuid.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := storage.GetURL(tt.filename)
			if got != tt.want {
				t.Errorf("GetURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLocalStorage_GenerateThumbnail(t *testing.T) {
	testDir := "./testdata/thumbnail-test"
	defer os.RemoveAll("./testdata")

	storage, err := NewLocalStorage(testDir, "http://localhost:8080/uploads")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name       string
		format     string
		width      int
		height     int
		wantErr    bool
		wantResize bool // Should image be resized?
	}{
		{
			name:       "generate thumbnail from large jpeg",
			format:     "jpeg",
			width:      800,
			height:     600,
			wantErr:    false,
			wantResize: true,
		},
		{
			name:       "generate thumbnail from large png",
			format:     "png",
			width:      1024,
			height:     768,
			wantErr:    false,
			wantResize: true,
		},
		{
			name:       "generate thumbnail from small image",
			format:     "jpeg",
			width:      100,
			height:     100,
			wantErr:    false,
			wantResize: false, // Should not resize as it's already small
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create and save test image
			imageData, err := createTestImage(tt.width, tt.height, tt.format)
			if err != nil {
				t.Fatalf("Failed to create test image: %v", err)
			}

			fileHeader, err := createFileHeader("test."+tt.format, imageData)
			if err != nil {
				t.Fatalf("Failed to create file header: %v", err)
			}

			originalFilename, err := storage.Save(ctx, fileHeader)
			if err != nil {
				t.Fatalf("Failed to save original file: %v", err)
			}

			// Generate thumbnail
			thumbnailFilename, err := storage.GenerateThumbnail(ctx, originalFilename)

			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateThumbnail() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify thumbnail filename
				if thumbnailFilename == "" {
					t.Error("GenerateThumbnail() returned empty filename")
					return
				}

				// Verify thumbnail contains "_thumb" suffix
				if !strings.Contains(thumbnailFilename, "_thumb") {
					t.Errorf("Thumbnail filename %s does not contain '_thumb'", thumbnailFilename)
				}

				// Verify thumbnail file exists
				thumbPath := filepath.Join(testDir, thumbnailFilename)
				if _, err := os.Stat(thumbPath); os.IsNotExist(err) {
					t.Errorf("Thumbnail file %s was not created", thumbPath)
					return
				}

				// Verify thumbnail is an image
				thumbFile, err := os.Open(thumbPath)
				if err != nil {
					t.Fatalf("Failed to open thumbnail: %v", err)
				}
				defer thumbFile.Close()

				thumbImg, _, err := image.Decode(thumbFile)
				if err != nil {
					t.Errorf("Failed to decode thumbnail as image: %v", err)
					return
				}

				// Verify thumbnail dimensions
				bounds := thumbImg.Bounds()
				thumbWidth := bounds.Dx()
				thumbHeight := bounds.Dy()

				// Thumbnail should be <= 300x300
				if thumbWidth > 300 || thumbHeight > 300 {
					t.Errorf("Thumbnail dimensions %dx%d exceed maximum 300x300", thumbWidth, thumbHeight)
				}

				// If original was larger, thumbnail should be smaller
				if tt.wantResize {
					if thumbWidth >= tt.width && thumbHeight >= tt.height {
						t.Errorf("Thumbnail (%dx%d) is not smaller than original (%dx%d)",
							thumbWidth, thumbHeight, tt.width, tt.height)
					}
				}
			}
		})
	}
}

func TestLocalStorage_GenerateThumbnail_Errors(t *testing.T) {
	testDir := "./testdata/thumbnail-error-test"
	defer os.RemoveAll("./testdata")

	storage, err := NewLocalStorage(testDir, "http://localhost:8080/uploads")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	ctx := context.Background()

	tests := []struct {
		name     string
		filename string
		wantErr  bool
	}{
		{
			name:     "non-existent file",
			filename: "nonexistent.jpg",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := storage.GenerateThumbnail(ctx, tt.filename)

			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateThumbnail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResizeImage(t *testing.T) {
	tests := []struct {
		name      string
		origWidth int
		origHeight int
		maxWidth  int
		maxHeight int
		wantLarger bool // Should result be larger than max?
	}{
		{
			name:       "resize large landscape image",
			origWidth:  800,
			origHeight: 600,
			maxWidth:   300,
			maxHeight:  300,
			wantLarger: false,
		},
		{
			name:       "resize large portrait image",
			origWidth:  600,
			origHeight: 800,
			maxWidth:   300,
			maxHeight:  300,
			wantLarger: false,
		},
		{
			name:       "keep small image unchanged",
			origWidth:  100,
			origHeight: 100,
			maxWidth:   300,
			maxHeight:  300,
			wantLarger: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test image
			img := image.NewRGBA(image.Rect(0, 0, tt.origWidth, tt.origHeight))

			// Resize
			result := resizeImage(img, tt.maxWidth, tt.maxHeight)

			// Check dimensions
			bounds := result.Bounds()
			width := bounds.Dx()
			height := bounds.Dy()

			if tt.wantLarger {
				// Should not exceed max dimensions
				if width <= tt.maxWidth && height <= tt.maxHeight {
					t.Errorf("Expected dimensions to exceed %dx%d, got %dx%d",
						tt.maxWidth, tt.maxHeight, width, height)
				}
			} else {
				// Should not exceed max dimensions
				if width > tt.maxWidth || height > tt.maxHeight {
					t.Errorf("Dimensions %dx%d exceed maximum %dx%d",
						width, height, tt.maxWidth, tt.maxHeight)
				}
			}

			// Verify aspect ratio is maintained (within reasonable tolerance)
			origAspect := float64(tt.origWidth) / float64(tt.origHeight)
			newAspect := float64(width) / float64(height)
			aspectDiff := origAspect - newAspect
			if aspectDiff < 0 {
				aspectDiff = -aspectDiff
			}

			// Allow 5% tolerance for aspect ratio
			tolerance := 0.05
			if aspectDiff > tolerance {
				t.Errorf("Aspect ratio not maintained: original=%.2f, new=%.2f",
					origAspect, newAspect)
			}
		})
	}
}
