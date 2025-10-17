# gemini-go-workshop
Workshop to discover Gemini 2.0 using the Go SDK

## Google AI API key

Create an API key at https://aistudio.google.com/apikey

## Google Cloud project

Create a project at https://console.cloud.google.com/

## Prerequistes

### Go 1.23+

Type `go version` to make sure.

### For samples requiring API key

```
export GOOGLE_GENAI_USE_VERTEXAI=false
export GOOGLE_API_KEY=<YOUR_API_KEY>
```

### For samples requiring Google Cloud Vertex AI

```
export GOOGLE_GENAI_USE_VERTEXAI=true
export GOOGLE_CLOUD_PROJECT=deleplace-ai-demos
export GOOGLE_CLOUD_LOCATION=us-central1
gcloud auth application-default login
gcloud services enable aiplatform.googleapis.com
```

## Read the samples

Open in your editor the source files `sample*.go`

## Run the samples

### Sample 0: Text input
```
go run . -n=0
```

Uncomment the code section to discover the structured response from the model.

Exercise: ask the same question but in French.

### Sample 1: Streaming text output
```
go run . -n=1
```

### Sample 2: Multimodal image input
```
go run . -n=2
```

Exercise: Is the last answer good enough? What happens if you use the model "gemini-2.5-pro" instead?

### Sample 3: Multimodal audio input
```
go run . -n=3
```

### Sample 4: Multimodal video input
```
go run . -n=4
```

Exercise: instead of a text question, provide the audio file ./testdata/question_about_video.mp3 as the question.

### Sample 5: Image generation
```
go run . -n=5
```
Exercise: write an extremely specific prompt to generate an image.

Note that safety filters are very sensitive and often refuse harmless prompts.

### Sample 6: Image upscaling
```
go run . -n=6
```

### Sample 7: Live streaming server
```
go run . -n=7
```

Open your browser at [http://localhost:8080](http://localhost:8080).

Click "Start Audio Conversation" and starting talking.

Click "Share screen" and ask questions about what Gemini sees on your screen.

Explore the source code and think about a use case for integrating these live capabilities in your application.

### Sample 8: Forbidden Words game
```
go run . -n=8
```

Open your browser at [http://localhost:8080](http://localhost:8080).
