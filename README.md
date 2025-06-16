# AWS Lambda Function for Ideogram Generation

This project provides an AWS Lambda function written in Go that integrates with the [Ideogram API](https://developer.ideogram.ai/api-reference/api-reference/generate-v3). The function receives a JSON request from an HTTP client (via a Lambda Function URL), sends the request to the Ideogram endpoint, and returns the generated ideogram image data back to the client. The function supports different image resolutions, aspect ratios, and style options.

## Project Overview

The Lambda function expects a JSON request body with the following fields:

- **prompt**: The text prompt for ideogram generation.
- **resolution**: The resolution of the generated image.
- **aspect_ratio**: The aspect ratio of the generated image.
- **num_images**: The number of images to generate.
- **style_type**: The style type for the ideogram generation.

The function will return the generated ideogram images in the response.

### Environment Variable

You must set the `API_KEY` environment variable in your Lambda function configuration. This key is required to authenticate requests to the **Ideogram API**.

## Steps to Get Started

### Prerequisites

Before you start, ensure you have the following installed and configured:

- **Go**: The Go programming language is required to compile and deploy the Lambda function. You can download and install Go from [https://golang.org/dl/](https://golang.org/dl/).
- **AWS CLI**: Make sure you have the AWS CLI installed and configured with the appropriate credentials. You can download and configure AWS CLI from [https://aws.amazon.com/cli/](https://aws.amazon.com/cli/).
- **AWS CloudFormation**: CloudFormation will be used to deploy resources such as Lambda functions, IAM roles, and API Gateway.

> [!NOTE] 
> If you are using cloudformation to deploy your lambda function then you would need to update the API_KEY under line number 49 of the aws_cloudformation_script.yaml file. Also, update line 52 with appropriate name of the zip file. To generate the artifact execute step 2 from below.

### Step 1: Set Up the API Key

In order to authenticate your requests to the **Ideogram API**, you will need to add an **API_KEY** environment variable to your Lambda function configuration.

1. Log into your AWS Management Console and navigate to the **Lambda** service.
2. Create a new Lambda function or use an existing one.
3. In the **Environment variables** section, add the environment variable:
   - **Key**: `API_KEY`
   - **Value**: Your Ideogram API key

### Step 2: Upload the Go Code to Lambda

1. Clone this repository to your local machine.
2. Compile the Go code:
   ```bash
   GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bootstrap main.go
   zip function.zip bootstrap
   ```
## Testing the Lambda Function using Zapier

1. Once the stack is deployed, obtain the API Gateway URL that was created during the CloudFormation stack deployment.
2. You can now send a POST request to this URL with a JSON payload containing the ideogram generation parameters (prompt, resolution, aspect_ratio, etc.).

### Example POST Request
```
{
  "prompt": "A futuristic cityscape",
  "resolution": "1024x1024",
  "aspect_ratio": "16:9",
  "num_images": 1,
  "style_type": "Sci-Fi"
}
```

3. The function will process the request, generate the ideogram image(s), and return the result.

### Example Response 
```
{
  "created": "2000-01-23 04:56:07+00:00",
  "data": [
    {
      "prompt": "A photo of a cat sleeping on a couch.",
      "resolution": "1024x1024",
      "is_image_safe": true,
      "seed": 12345,
      "url": "https://ideogram.ai/api/images/ephemeral/xtdZiqPwRxqY1Y7NExFmzB.png?exp=1743867804&sig=e13e12677633f646d8531a153d20e2d3698dca9ee7661ee5ba4f3b64e7ec3f89",
      "style_type": "GENERAL"
    }
  ]
}
```


