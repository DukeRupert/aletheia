package main

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"

	_ "image/jpeg" // Register JPEG decoder
	_ "image/png"  // Register PNG decoder
)

// resizeImage resizes an image to fit within maxWidth x maxHeight while maintaining aspect ratio
func resizeImage(img image.Image, maxWidth, maxHeight int) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// If image is already smaller than max dimensions, return as-is
	if width <= maxWidth && height <= maxHeight {
		return img
	}

	// Calculate scaling factor to fit within maxWidth x maxHeight
	scaleX := float64(maxWidth) / float64(width)
	scaleY := float64(maxHeight) / float64(height)
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	newWidth := int(float64(width) * scale)
	newHeight := int(float64(height) * scale)

	// Create new image with target dimensions
	resized := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Nearest-neighbor resampling
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			srcX := int(float64(x) / scale)
			srcY := int(float64(y) / scale)
			resized.Set(x, y, img.At(srcX, srcY))
		}
	}

	return resized
}

// cropToAspectRatio crops image to specific aspect ratio from center
func cropToAspectRatio(img image.Image, aspectWidth, aspectHeight int) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	targetRatio := float64(aspectWidth) / float64(aspectHeight)
	currentRatio := float64(width) / float64(height)

	var cropX, cropY, cropWidth, cropHeight int

	if currentRatio > targetRatio {
		// Image is too wide, crop width
		cropHeight = height
		cropWidth = int(float64(height) * targetRatio)
		cropX = (width - cropWidth) / 2
		cropY = 0
	} else {
		// Image is too tall, crop height
		cropWidth = width
		cropHeight = int(float64(width) / targetRatio)
		cropX = 0
		cropY = (height - cropHeight) / 2
	}

	cropped := image.NewRGBA(image.Rect(0, 0, cropWidth, cropHeight))
	for y := 0; y < cropHeight; y++ {
		for x := 0; x < cropWidth; x++ {
			cropped.Set(x, y, img.At(cropX+x, cropY+y))
		}
	}

	return cropped
}

func main() {
	// Source image
	sourceFile := "internal/testdata/images/medium/IMG_0612.JPG"

	// Open source image
	file, err := os.Open(sourceFile)
	if err != nil {
		fmt.Printf("Error opening source image: %v\n", err)
		return
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		fmt.Printf("Error decoding image: %v\n", err)
		return
	}

	fmt.Printf("Source image dimensions: %dx%d\n", img.Bounds().Dx(), img.Bounds().Dy())

	// Generate variants
	variants := []struct {
		path   string
		img    image.Image
		format string
	}{
		// Small versions (300x225)
		{
			path:   "internal/testdata/images/small/test_small.jpg",
			img:    resizeImage(img, 300, 225),
			format: "jpeg",
		},
		{
			path:   "internal/testdata/images/small/test_small.png",
			img:    resizeImage(img, 300, 225),
			format: "png",
		},

		// Large version (keep original size or slightly larger)
		{
			path:   "internal/testdata/images/large/test_large.jpg",
			img:    img, // Keep original
			format: "jpeg",
		},

		// Format variants
		{
			path:   "internal/testdata/images/formats/test.jpg",
			img:    img,
			format: "jpeg",
		},
		{
			path:   "internal/testdata/images/formats/test.png",
			img:    img,
			format: "png",
		},

		// Edge cases
		{
			path:   "internal/testdata/images/edge_cases/test_tiny.jpg",
			img:    resizeImage(img, 10, 10),
			format: "jpeg",
		},
		{
			path:   "internal/testdata/images/edge_cases/test_portrait.jpg",
			img:    resizeImage(cropToAspectRatio(img, 3, 4), 300, 400),
			format: "jpeg",
		},
		{
			path:   "internal/testdata/images/edge_cases/test_landscape.jpg",
			img:    resizeImage(cropToAspectRatio(img, 16, 9), 640, 360),
			format: "jpeg",
		},
	}

	// Save all variants
	for _, v := range variants {
		if err := os.MkdirAll(filepath.Dir(v.path), 0755); err != nil {
			fmt.Printf("Error creating directory for %s: %v\n", v.path, err)
			continue
		}

		outFile, err := os.Create(v.path)
		if err != nil {
			fmt.Printf("Error creating file %s: %v\n", v.path, err)
			continue
		}

		switch v.format {
		case "jpeg":
			err = jpeg.Encode(outFile, v.img, &jpeg.Options{Quality: 85})
		case "png":
			err = png.Encode(outFile, v.img)
		}

		outFile.Close()

		if err != nil {
			fmt.Printf("Error encoding %s: %v\n", v.path, err)
		} else {
			info, _ := os.Stat(v.path)
			fmt.Printf("✓ Created %s (%dx%d, %d bytes)\n",
				v.path,
				v.img.Bounds().Dx(),
				v.img.Bounds().Dy(),
				info.Size())
		}
	}

	fmt.Println("\nTest images generated successfully!")
}
