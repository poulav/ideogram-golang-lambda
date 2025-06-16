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
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type FreepikResponse struct {
	Original       string `json:"original,omitempty"`
	HighResolution string `json:"high_resolution,omitempty"`
	Preview        string `json:"preview,omitempty"`
	URL            string `json:"url,omitempty"`
}

type ColourPalette struct {
	Members []struct {
		ColorHex    string  `json:"color_hex"`
		ColorWeight *string `json:"color_weight,omitempty"`
	} `json:"members,omitempty"`
}

type IdeogramRequestBody struct {
	Prompt        string         `json:"prompt"`
	FileName      string         `json:"filename"`
	Resolution    *string        `json:"resolution,omitempty"`
	AspectRatio   *string        `json:"aspect_ratio,omitempty"`
	NumImages     *int           `json:"num_images,omitempty"`
	StyleType     *string        `json:"style_type,omitempty"`
	ColourPalette *ColourPalette `json:"colour_palette,omitempty"`
}

type IdeogramResponse struct {
	Created string `json:"created"`
	Data    []struct {
		Prompt      string `json:"prompt"`
		Resolution  string `json:"resolution"`
		IsImageSafe bool   `json:"is_image_safe"`
		Seed        int    `json:"seed"`
		URL         string `json:"url"`
		StyleType   string `json:"style_type"`
	} `json:"data"`
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
	} else {
		decodedBody = []byte(body)
	}
	log.Println("Decoded body:", string(decodedBody))
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

	// After getting the response, download the image and send it to Freepik API
	var ideogramResponse IdeogramResponse
	err = json.Unmarshal([]byte(response), &ideogramResponse)
	if err != nil {
		log.Println("Error unmarshalling ideogram response:", err)
		return events.LambdaFunctionURLResponse{
			StatusCode: 500,
			Body:       "Internal Server Error",
		}, nil
	}

	s3URLs := make([]string, 0)
	for i := range ideogramResponse.Data {
		// Assuming there's only one image in the response
		imageURL := ideogramResponse.Data[i].URL
		log.Println("Image URL from Ideogram:", imageURL)

		// Download the image
		imageData, err := downloadImage(imageURL)
		if err != nil {
			log.Println("Error downloading image:", err)
			return events.LambdaFunctionURLResponse{
				StatusCode: 500,
				Body:       "Error downloading image",
			}, nil
		}

		// Upload the image to S3
		s3URL, err := uploadImageToS3(imageData, ideogramRequestBody.FileName)
		if err != nil {
			log.Println("Error uploading image to S3:", err)
			return events.LambdaFunctionURLResponse{
				StatusCode: 500,
				Body:       "Error uploading image to S3",
			}, nil
		}
		log.Println("Ideogram Image uploaded to S3:", s3URL)

		// Remove Background via Freepik
		response, err := removeImageBGviaFreepik(s3URL)
		if err != nil {
			log.Println("Error removing image background:", err)
			return events.LambdaFunctionURLResponse{
				StatusCode: 500,
				Body:       "Error removing image background",
			}, nil
		}

		// After getting the response from Freepik, download the image and upload it to S3
		var freepikResponse FreepikResponse
		err = json.Unmarshal([]byte(response), &freepikResponse)
		if err != nil {
			log.Println("Error unmarshalling ideogram response:", err)
			return events.LambdaFunctionURLResponse{
				StatusCode: 500,
				Body:       "Internal Server Error",
			}, nil
		}

		log.Println("Freepik response:", freepikResponse.URL)

		// Download the Freepik image
		freepikImage, err := downloadImage(freepikResponse.URL)
		if err != nil {
			log.Println("Error downloading image:", err)
			return events.LambdaFunctionURLResponse{
				StatusCode: 500,
				Body:       "Error downloading image",
			}, nil
		}

		// Upload the image to S3
		fs3URL, err := uploadImageToS3(freepikImage, ideogramRequestBody.FileName)
		if err != nil {
			log.Println("Error uploading image to S3:", err)
			return events.LambdaFunctionURLResponse{
				StatusCode: 500,
				Body:       "Error uploading image to S3",
			}, nil
		}
		log.Println("Freepik Image uploaded to S3:", fs3URL)

		s3URLs = append(s3URLs, fs3URL)
	}

	responseBody, err := json.Marshal(map[string][]string{"image_urls": s3URLs})
	if err != nil {
		return events.LambdaFunctionURLResponse{
			StatusCode: 500,
			Body:       "Error marshaling response",
		}, nil
	}
	return events.LambdaFunctionURLResponse{
		StatusCode: 200,
		Body:       string(responseBody),
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
	if body.Resolution != nil {
		writer.WriteField("resolution", *body.Resolution)
	} else if body.AspectRatio != nil {
		writer.WriteField("aspect_ratio", *body.AspectRatio)
	}
	if body.NumImages != nil {
		writer.WriteField("num_images", fmt.Sprintf("%d", *body.NumImages))
	}
	if body.StyleType != nil {
		writer.WriteField("style_type", *body.StyleType)
	}
	if body.ColourPalette != nil {
		for i, member := range body.ColourPalette.Members {
			memberPrefix := fmt.Sprintf("colour_palette[members][%d]", i)
			writer.WriteField(memberPrefix+"[color_hex]", member.ColorHex)
			if member.ColorWeight != nil {
				writer.WriteField(memberPrefix+"[color_weight]", *member.ColorWeight)
			}
		}
	}

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

// Download the image from the URL
func downloadImage(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching image: %v", err)
	}
	defer resp.Body.Close()

	// Read the image data
	imageData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading image data: %v", err)
	}

	return imageData, nil
}

// Upload the image to S3
func uploadImageToS3(imageData []byte, filename string) (string, error) {
	bucket_name := os.Getenv("BUCKET_NAME")

	if bucket_name == "" {
		return "", fmt.Errorf("BUCKET_NAME is not set")
	}
	folder_name := os.Getenv("FOLDER_NAME")

	if folder_name == "" {
		return "", fmt.Errorf("FOLDER_NAME is not set")
	}
	bucket_region := os.Getenv("BUCKET_REGION")

	if bucket_region == "" {
		return "", fmt.Errorf("BUCKET_REGION is not set")
	}

	// Create an S3 session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(bucket_region),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create session: %v", err)
	}

	// Create an S3 service client
	s3Svc := s3.New(sess)

	// Set the bucket and key (file name)
	key := folder_name + "/" + filename + ".png"

	// Upload the image
	_, err = s3Svc.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(bucket_name),
		Key:         aws.String(key),
		Body:        bytes.NewReader(imageData),
		ContentType: aws.String("image/png"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload image: %v", err)
	}

	// Return the S3 URL
	s3URL := fmt.Sprintf("https://%s.s3.amazonaws.com/%s", bucket_name, key)
	return s3URL, nil
}

func removeImageBGviaFreepik(imageUrl string) (string, error) {

	url := "https://api.freepik.com/v1/ai/beta/remove-background"

	payload := strings.NewReader("image_url=" + imageUrl)

	req, _ := http.NewRequest("POST", url, payload)

	req.Header.Add("x-freepik-api-key", os.Getenv("FREEPIK_API_KEY"))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request to Freepik: %v", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	// fmt.Println(res)
	fmt.Println(string(body))

	return string(body), nil
}

func main() {
	lambda.Start(handleRequest)
}
