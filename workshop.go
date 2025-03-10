package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"google.golang.org/genai"
)

var N = flag.Int("n", -1, "index of the sample to run")

var client *genai.Client

func main() {
	flag.Parse()
	flag.Usage = usage

	if *N < 0 || *N >= len(samples) {
		usage()
		return
	}
	sample := samples[*N]

	ctx := context.Background()

	//
	// Create the Gemini client
	//
	var err error
	for _, k := range []string{
		"GOOGLE_API_KEY",
		"GOOGLE_GENAI_USE_VERTEXAI",
		"GOOGLE_CLOUD_PROJECT",
		"GOOGLE_CLOUD_LOCATION",
	} {
		fmt.Printf("%s=%s\n", k, os.Getenv(k))
	}
	client, err = genai.NewClient(ctx, &genai.ClientConfig{
		// empty ClientConfig automatically uses the env vars listed above
	})
	if err != nil {
		log.Fatal(err)
	}
	if client.ClientConfig().Backend == genai.BackendVertexAI {
		fmt.Println("(using VertexAI backend)")
	} else {
		fmt.Println("(using GeminiAPI backend)")
	}
	fmt.Println()

	//
	// Run the selected sample
	//
	err = sample.f(ctx)
	if err != nil {
		log.Fatal(err)
	}
}

var samples = []namedSample{
	0: {name: "Text prompt, text answer", f: sample0_text},
	1: {name: "Text prompt, streaming text output", f: sample1_text_stream},
	2: {name: "Multimodal prompt: text and image", f: sample2_imageInput},
	3: {name: "Multimodal prompt: audio", f: sample3_audioInput},
	4: {name: "Multimodal prompt: video", f: sample4_videoInput},
}

func usage() {
	fmt.Fprintln(os.Stderr, "Syntax:\n\tgo run . -n=N")
	fmt.Fprintf(os.Stderr, "\nwhere N is the index of a sample:\n\n")
	for i, s := range samples {
		fmt.Fprintf(os.Stderr, "\t%d\t%s\n", i, s.name)
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "To use the GeminiAPI backend, set the GOOGLE_API_KEY env var.")

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, "To use the VertexAI backend, set the GOOGLE_GENAI_USE_VERTEXAI, GOOGLE_CLOUD_PROJECT, GOOGLE_CLOUD_LOCATION env vars.")
	fmt.Fprintln(os.Stderr)

	os.Exit(1)
}

type namedSample struct {
	name string
	f    func(context.Context) error
}

func checkResponse(res *genai.GenerateContentResponse, err error) {
	if err != nil {
		log.Fatal(err)
	}
	checkNotEmpty(res)
}

func checkNotEmpty(res *genai.GenerateContentResponse) {
	if len(res.Candidates) == 0 ||
		len(res.Candidates[0].Content.Parts) == 0 {
		log.Fatalf("empty response from model")
	}
}

func textOf(res *genai.GenerateContentResponse) string {
	checkNotEmpty(res)
	return res.Candidates[0].Content.Parts[0].Text
}

func printTextResponse(res *genai.GenerateContentResponse) {
	fmt.Println(textOf(res))
}
