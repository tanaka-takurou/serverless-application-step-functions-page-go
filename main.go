package main

import (
	"io"
	"os"
	"log"
	"bytes"
	"embed"
	"context"
	"net/http"
	"html/template"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type TemplateData struct {
	Title   string
	ApiPath string
	Bucket  string
}

//go:embed templates
var templateFS embed.FS

func HandleRequest(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayProxyResponse, error) {
	var dat TemplateData
	buf := new(bytes.Buffer)
	fw := io.Writer(buf)
	tmp := template.Must(template.New("tmp").ParseFS(templateFS, "templates/index.html", "templates/header.html", "templates/view.html"))
	dat.Title = "Step Functions"
	dat.ApiPath = os.Getenv("API_PATH")
	dat.Bucket = "https://" + os.Getenv("BUCKET_NAME") + ".s3-" + os.Getenv("REGION") + ".amazonaws.com/"
	if err := tmp.ExecuteTemplate(fw, "base", dat); err != nil {
		log.Fatal(err)
	}
	return events.APIGatewayProxyResponse{
		StatusCode:      http.StatusOK,
		IsBase64Encoded: false,
		Body:            string(buf.Bytes()),
		Headers: map[string]string{
			"Content-Type": "text/html",
		},
	}, nil
}

func main() {
	lambda.Start(HandleRequest)
}
