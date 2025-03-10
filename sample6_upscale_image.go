package main

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/genai"
)

// To run this sample with a VertexAI Google Cloud project:
//
// $ export GOOGLE_GENAI_USE_VERTEXAI=true
// $ export GOOGLE_CLOUD_PROJECT=xxxxxxxxx
// $ export GOOGLE_CLOUD_LOCATION=us-central1
// $ gcloud auth application-default login
// $ gcloud services enable aiplatform.googleapis.com
// $ go run . -n=6

func sample6_upscaleImage(ctx context.Context) error {
	modelName := "imagen-3.0-generate-002"

	imgdata, err := os.ReadFile("./testdata/lion.jpg")
	if err != nil {
		return err
	}

	var config *genai.UpscaleImageConfig = &genai.UpscaleImageConfig{
		OutputMIMEType:   "image/jpeg",
		IncludeRAIReason: true,
	}
	image := &genai.Image{
		ImageBytes: imgdata,
		MIMEType:   "image/jpeg",
	}
	result, err := client.Models.UpscaleImage(ctx, modelName, image, "x4", config)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("upscaled_image.jpg")
	fmt.Println("Writing file", path)
	err = os.WriteFile(path, result.GeneratedImages[0].Image.ImageBytes, 0777)
	if err != nil {
		return err
	}

	return nil
}
