package main

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/genai"
)

// To run this sample with a Gemini API key:
//
// $ export GOOGLE_API_KEY=xxxxxxxxxx
// $ go run . -n=3

func sample3_audioInput(ctx context.Context) error {
	modelName := "gemini-2.0-flash-001"

	// Load an audio file to create a multimodal prompt
	path := "./testdata/math.m4a"
	audiodata, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	fmt.Println("Input: ", path)
	fmt.Println()

	prompt := []*genai.Content{
		{
			Parts: []*genai.Part{
				genai.NewPartFromBytes(audiodata, "audio/mp4"),
			},
		},
	}
	result, err := client.Models.GenerateContent(ctx, modelName, prompt, nil)
	if err != nil {
		return err
	}
	answer := textOf(result)
	fmt.Println("Answer:", answer)
	fmt.Println()

	return nil
}
