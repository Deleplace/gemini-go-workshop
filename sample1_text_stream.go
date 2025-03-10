package main

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

// To run this sample with a Gemini API key:
//
// $ export GOOGLE_API_KEY=xxxxxxxxxx
// $ go run . -n=1

func sample1_textStream(ctx context.Context) error {
	modelName := "gemini-2.0-flash-001"
	prompt := "Tell me a story in 300 words."
	fmt.Println("Prompt:", prompt)
	fmt.Println()

	// GenerateContentStream returns an iterator of type iter.Seq2[*GenerateContentResponse, error].
	// The package iter was introduced in Go 1.23, thus the genai package requires min Go 1.23.
	iterator := client.Models.GenerateContentStream(ctx, modelName, genai.Text(prompt), nil)

	for result, err := range iterator {
		if err != nil {
			return err
		}
		fmt.Print(textOf(result))
	}

	return nil
}
