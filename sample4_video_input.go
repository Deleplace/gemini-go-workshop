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
// $ go run . -n=4

func sample4_videoInput(ctx context.Context) error {
	modelName := "gemini-2.5-flash-lite"

	// Load a video file to create a multimodal prompt
	videodata, err := os.ReadFile("./testdata/pixel8.mp4")
	if err != nil {
		return err
	}

	question1 := "How many people are in this video?"
	question2 := "In which country was this video filmed?"
	question3 := "Are there animals in this video?"
	fmt.Println("Question 1:", question1)
	fmt.Println("Question 2:", question2)
	fmt.Println("Question 3:", question3)
	fmt.Println()

	multimodalPrompt := []*genai.Content{
		{
			Parts: []*genai.Part{
				genai.NewPartFromBytes(videodata, "video/mp4"),
				genai.NewPartFromText(question1),
				genai.NewPartFromText(question2),
				genai.NewPartFromText(question3),
			},
		},
	}
	result, err := client.Models.GenerateContent(ctx, modelName, multimodalPrompt, nil)
	if err != nil {
		return err
	}
	answers := textOf(result)
	fmt.Println("Answers:", answers)
	fmt.Println()

	//
	// Exercise:
	// instead of a text question, provide the audio file ./testdata/question_about_video.mp3 as the question.
	//

	return nil
}
