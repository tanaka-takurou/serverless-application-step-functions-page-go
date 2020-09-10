package main

import (
	"os"
	"log"
	"time"
	"bytes"
	"errors"
	"strings"
	"context"
	"net/http"
	"path/filepath"
	"encoding/json"
	"encoding/base64"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3manager"
)

type APIResponse struct {
	Message  string `json:"message"`
}

type Response events.APIGatewayProxyResponse

var cfg aws.Config
var s3Client *s3.Client
var sfnClient *sfn.Client

const layout  string = "2006-01-02-15-04"
const layout2 string = "20060102150405.000"

func HandleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (Response, error) {
	var jsonBytes []byte
	var err error
	d := make(map[string]string)
	json.Unmarshal([]byte(request.Body), &d)
	if v, ok := d["action"]; ok {
		switch v {
		case "upload" :
			if v, ok := d["filename"]; ok {
				if w, ok := d["filedata"]; ok {
					if name, key, e := uploadImage(v, w); e == nil {
						err = startExecution(ctx, name, key)
						if err == nil {
							jsonBytes, _ = json.Marshal(APIResponse{Message: name})
						}
					} else {
						err = e
					}
				}
			}
		case "checkstatus" :
			if id, ok := d["id"]; ok {
				res, e := checkStatus(ctx, id)
				if e != nil {
					err = e
				} else {
					jsonBytes, _ = json.Marshal(APIResponse{Message: res})
				}
			}
		}
	}
	if err != nil {
		return Response{
			StatusCode: http.StatusInternalServerError,
		}, err
	} else {
		log.Print(request.RequestContext.Identity.SourceIP)
	}
	responseBody := ""
	if len(jsonBytes) > 0 {
		responseBody = string(jsonBytes)
	}
	return Response {
		StatusCode: http.StatusOK,
		Body: responseBody,
	}, nil
}

func uploadImage(filename string, filedata string)(string, string, error) {
	t := time.Now()
	b64data := filedata[strings.IndexByte(filedata, ',')+1:]
	data, err := base64.StdEncoding.DecodeString(b64data)
	if err != nil {
		log.Print(err)
		return "", "", err
	}
	extension := filepath.Ext(filename)
	var contentType string

	switch extension {
	case ".jpg":
		contentType = "image/jpeg"
	case ".jpeg":
		contentType = "image/jpeg"
	case ".gif":
		contentType = "image/gif"
	case ".png":
		contentType = "image/png"
	default:
		return "", "", errors.New("this extension is invalid")
	}
	name := strings.Replace(t.Format(layout2), ".", "", 1)
	key := strings.Replace(t.Format(layout), ".", "", 1) + "/" + name + extension
	uploader := s3manager.NewUploader(cfg)
	_, err = uploader.Upload(&s3manager.UploadInput{
		ACL: s3.ObjectCannedACLPublicRead,
		Bucket: aws.String(os.Getenv("BUCKET_NAME")),
		Key: aws.String(key),
		Body: bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		log.Print(err)
		return "", "", err
	}
	return name, key, nil
}

func startExecution(ctx context.Context, name string, key string) error {
	if sfnClient == nil {
		sfnClient = sfn.New(cfg)
	}
	input := &sfn.StartExecutionInput{
		Input: aws.String("{\"Key\" : \"" + key + "\"}"),
		Name: aws.String(name),
		StateMachineArn: aws.String(os.Getenv("STATE_MACHINE_ARN")),
	}

	req := sfnClient.StartExecutionRequest(input)
	_, err := req.Send(ctx)
	if err != nil {
		log.Print(err)
		return err
	}
	return nil
}

func checkStatus(ctx context.Context, id string)(string, error) {
	if sfnClient == nil {
		sfnClient = sfn.New(cfg)
	}

	statusList := []sfn.ExecutionStatus{sfn.ExecutionStatusRunning, sfn.ExecutionStatusSucceeded}

	for _, v := range statusList {
		input := &sfn.ListExecutionsInput{
			StateMachineArn: aws.String(os.Getenv("STATE_MACHINE_ARN")),
			StatusFilter: v,
		}

		req := sfnClient.ListExecutionsRequest(input)
		res, err := req.Send(ctx)
		if err != nil {
			log.Print(err)
			return "", err
		}
		for _, w := range res.ListExecutionsOutput.Executions {
			if id == aws.StringValue(w.Name) {
				return string(v), nil
			}
		}
	}

	return "Error", nil
}

func init() {
	var err error
	cfg, err = external.LoadDefaultAWSConfig()
	cfg.Region = os.Getenv("REGION")
	if err != nil {
		log.Print(err)
	}
}

func main() {
	lambda.Start(HandleRequest)
}
