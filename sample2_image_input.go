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
// $ go run . -n=2

func sample2_imageInput(ctx context.Context) error {
	modelName := "gemini-2.0-flash-001"

	// Is the last answer good enough?
	// What happens if you use the model "gemini-2.0-flash-thinking-exp-01-21" instead?

	// Load an image to create a multimodal prompt
	imgdata, err := os.ReadFile("./testdata/pool.png")
	if err != nil {
		return err
	}

	question1 := "Describe this image"
	fmt.Println("Question:", question1)
	fmt.Println()

	multimodalPrompt1 := []*genai.Content{
		{
			Parts: []*genai.Part{
				genai.NewPartFromBytes(imgdata, "image/png"),
				genai.NewPartFromText(question1),
			},
		},
	}
	result1, err := client.Models.GenerateContent(ctx, modelName, multimodalPrompt1, nil)
	if err != nil {
		return err
	}
	answer1 := textOf(result1)
	fmt.Println("Answer:", answer1)
	fmt.Println()

	question2 := "How do I use three of the pool balls in this image to sum up to 30?"
	fmt.Println("Question:", question2)
	fmt.Println()

	multimodalPrompt2 := []*genai.Content{
		{
			Parts: []*genai.Part{
				genai.NewPartFromBytes(imgdata, "image/png"),
				genai.NewPartFromText(question2),
			},
		},
	}
	result2, err := client.Models.GenerateContent(ctx, modelName, multimodalPrompt2, nil)
	if err != nil {
		return err
	}
	answer2 := textOf(result2)
	fmt.Println("Answer:", answer2)
	fmt.Println()

	return nil
}
