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
// $ go run . -n=5

func sample5_generateImage(ctx context.Context) error {
	modelName := "imagen-3.0-generate-002"
	prompt := "Create an overly decorated umbrella."
	fmt.Println("Prompt:", prompt)
	fmt.Println()

	var n int32 = 4
	var config *genai.GenerateImagesConfig = &genai.GenerateImagesConfig{
		NumberOfImages:   &n,
		OutputMIMEType:   "image/jpeg",
		IncludeRAIReason: true,
	}
	result, err := client.Models.GenerateImages(ctx, modelName, prompt, config)
	if err != nil {
		return err
	}

	for i, img := range result.GeneratedImages {
		path := fmt.Sprintf("generated_image_%d.jpg", i)
		fmt.Println("Writing image to file", path)
		err := os.WriteFile(path, img.Image.ImageBytes, 0777)
		if err != nil {
			return err
		}
	}

	//
	// Exercise:
	// write an extremely specific prompt to generate an image.
	//
	// Note that safety filters are very sensitive and often refuse harmless prompts.

	return nil
}
