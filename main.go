// Sending Request to ideogram endpoint and reverting back the response to Zapier
// This Lambda function is triggered by a Lambda Function URL and sends a request to the ideogram endpoint.
// The response from the ideogram endpoint is then returned to the caller.
// The function uses the AWS Lambda Go SDK and the net/http package to handle HTTP requests and responses.
// It also uses the encoding/json package to handle JSON data and the encoding/base64 package to decode base64 encoded data.
// The function is designed to be deployed as an AWS Lambda function and is triggered by a Lambda Function URL.
// The function expects a JSON request body with the following fields:
// - prompt: The text prompt for the ideogram generation.
// - resolution: The resolution of the generated image.
// - aspect_ratio: The aspect ratio of the generated image.
// - num_images: The number of images to generate.
// - style_type: The style type for the ideogram generation.
// The function returns a JSON response with the generated ideogram images.

// You must add the API_KEY environment variable in your Lambda function configuration.
// The API_KEY is used to authenticate the request to the ideogram endpoint.
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type IdeogramRequestBody struct {
    Prompt      string `json:"prompt"`
    Resolution  string `json:"resolution"`
    AspectRatio string `json:"aspect_ratio"`
    NumImages   int    `json:"num_images"`
    StyleType   string `json:"style_type"`
}

func handleRequest(request events.LambdaFunctionURLRequest) (events.LambdaFunctionURLResponse, error) {

	// Extract the request body
	body := request.Body
	var ideogramRequestBody IdeogramRequestBody
	var decodedBody []byte
	var err error

	//For Zapier, the request body is base64 encoded
	if request.IsBase64Encoded {
		decodedBody, err = base64.StdEncoding.DecodeString(body)
		if err != nil {
			log.Println("Error decoding base64 body:", err)
			return events.LambdaFunctionURLResponse{
				StatusCode: 400,
				Body:       "Bad Request: invalid base64",
			}, nil
		} 
	}else {
			decodedBody = []byte(body)
	}

	err = json.Unmarshal(decodedBody, &ideogramRequestBody)
	if err != nil {
		log.Println("Error unmarshalling request body:", err)
		return events.LambdaFunctionURLResponse{
			StatusCode: 400,
			Body:       "Bad Request",
		}, nil
	}

	// Send the request to the ideogram endpoint and get the response
	response, err := sendRequestToIdeogram(ideogramRequestBody)
	if err != nil {
		log.Println("Error sending request to ideogram:", err)
		return events.LambdaFunctionURLResponse{
			StatusCode: 500,
			Body:       "Internal Server Error",
		}, nil
	}

	return events.LambdaFunctionURLResponse{
		StatusCode: 200,
		Body:       response,
	}, nil
}

func sendRequestToIdeogram(body IdeogramRequestBody) (string, error) {
	// Load environment variables from .env file
	api_key := os.Getenv("API_KEY")

	if api_key == "" {
		return "", fmt.Errorf("API_KEY is not set")
	}

	// Create a buffer and multipart writer
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add fields as per the API documentation
	writer.WriteField("prompt", body.Prompt)
	if body.Resolution != "" {
		writer.WriteField("resolution", body.Resolution)
	}else {
		writer.WriteField("aspect_ratio", body.AspectRatio)
	}	
	writer.WriteField("num_images", fmt.Sprintf("%d", body.NumImages))
	writer.WriteField("style_type", body.StyleType)

	writer.Close()

	// Make the request to the ideogram endpoint
	endpoint := "https://api.ideogram.ai/v1/ideogram-v3/generate"
	req, err := http.NewRequest("POST", endpoint, &buf)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Api-Key", api_key)

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()
	respBody := new(bytes.Buffer)
	respBody.ReadFrom(resp.Body)
	return respBody.String(), nil
}

func main() {
	lambda.Start(handleRequest)
}
