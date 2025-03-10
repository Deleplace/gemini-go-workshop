package main

import (
	"context"
	"fmt"

	"google.golang.org/genai"
)

// To run this sample with a Gemini API key:
//
// $ export GOOGLE_API_KEY=xxxxxxxxxx
// $ go run . -n=0

func sample0_text(ctx context.Context) error {
	modelName := "gemini-1.5-pro-002"
	question := "When was the battle of Austerlitz?"
	fmt.Println("Question:", question)

	result, err := client.Models.GenerateContent(ctx, modelName, genai.Text(question), nil)
	if err != nil {
		return err
	}

	// We expect the result to contain 1 candidate with 1 part
	answer := textOf(result)
	fmt.Println("Answer:", answer)

	// Uncomment this to discover the structured response from the model
	//
	// response, err := json.MarshalIndent(*result, "", "  ")
	// if err != nil {
	// 	return err
	// }
	// fmt.Println(string(response))

	return nil
}
