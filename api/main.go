package main

import (
	"os"
	"fmt"
	"log"
	"time"
	"bytes"
	"errors"
	"strings"
	"reflect"
	"context"
	"net/http"
	"path/filepath"
	"encoding/json"
	"encoding/base64"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sfn"
	ftypes "github.com/aws/aws-sdk-go-v2/service/sfn/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	stypes "github.com/aws/aws-sdk-go-v2/service/s3/types"
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
					if name, key, e := uploadImage(ctx, v, w); e == nil {
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

func uploadImage(ctx context.Context, filename string, filedata string)(string, string, error) {
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
	if s3Client == nil {
		s3Client = getS3Client()
	}
	input := &s3.PutObjectInput{
		ACL: stypes.ObjectCannedACLPublicRead,
		Bucket: aws.String(os.Getenv("BUCKET_NAME")),
		Key: aws.String(key),
		Body: bytes.NewReader(data),
		ContentType: aws.String(contentType),
	}
	_, err = s3Client.PutObject(ctx, input)
	if err != nil {
		log.Print(err)
		return "", "", err
	}
	return name, key, nil
}

func startExecution(ctx context.Context, name string, key string) error {
	if sfnClient == nil {
		sfnClient = getSfnClient()
	}
	input := &sfn.StartExecutionInput{
		Input: aws.String("{\"Key\" : \"" + key + "\"}"),
		Name: aws.String(name),
		StateMachineArn: aws.String(os.Getenv("STATE_MACHINE_ARN")),
	}

	_, err := sfnClient.StartExecution(ctx, input)
	if err != nil {
		log.Print(err)
		return err
	}
	return nil
}

func checkStatus(ctx context.Context, id string)(string, error) {
	if sfnClient == nil {
		sfnClient = getSfnClient()
	}

	statusList := []ftypes.ExecutionStatus{ftypes.ExecutionStatusRunning, ftypes.ExecutionStatusSucceeded}

	for _, v := range statusList {
		input := &sfn.ListExecutionsInput{
			StateMachineArn: aws.String(os.Getenv("STATE_MACHINE_ARN")),
			StatusFilter: v,
		}

		res, err := sfnClient.ListExecutions(ctx, input)
		if err != nil {
			log.Print(err)
			return "", err
		}
		for _, w := range res.Executions {
			if id == stringValue(w.Name) {
				return string(v), nil
			}
		}
	}

	return "Error", nil
}

func getSfnClient() *sfn.Client {
	if cfg.Region != os.Getenv("REGION") {
		cfg = getConfig()
	}
	return sfn.NewFromConfig(cfg)
}

func getS3Client() *s3.Client {
	if cfg.Region != os.Getenv("REGION") {
		cfg = getConfig()
	}
	return s3.NewFromConfig(cfg)
}

func getConfig() aws.Config {
	var err error
	newConfig, err := config.LoadDefaultConfig()
	newConfig.Region = os.Getenv("REGION")
	if err != nil {
		log.Print(err)
	}
	return newConfig
}

func stringValue(i interface{}) string {
	var buf bytes.Buffer
	strVal(reflect.ValueOf(i), 0, &buf)
	res := buf.String()
	return res[1:len(res) - 1]
}

func strVal(v reflect.Value, indent int, buf *bytes.Buffer) {
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Struct:
		buf.WriteString("{\n")
		for i := 0; i < v.Type().NumField(); i++ {
			ft := v.Type().Field(i)
			fv := v.Field(i)
			if ft.Name[0:1] == strings.ToLower(ft.Name[0:1]) {
				continue // ignore unexported fields
			}
			if (fv.Kind() == reflect.Ptr || fv.Kind() == reflect.Slice) && fv.IsNil() {
				continue // ignore unset fields
			}
			buf.WriteString(strings.Repeat(" ", indent+2))
			buf.WriteString(ft.Name + ": ")
			if tag := ft.Tag.Get("sensitive"); tag == "true" {
				buf.WriteString("<sensitive>")
			} else {
				strVal(fv, indent+2, buf)
			}
			buf.WriteString(",\n")
		}
		buf.WriteString("\n" + strings.Repeat(" ", indent) + "}")
	case reflect.Slice:
		nl, id, id2 := "", "", ""
		if v.Len() > 3 {
			nl, id, id2 = "\n", strings.Repeat(" ", indent), strings.Repeat(" ", indent+2)
		}
		buf.WriteString("[" + nl)
		for i := 0; i < v.Len(); i++ {
			buf.WriteString(id2)
			strVal(v.Index(i), indent+2, buf)
			if i < v.Len()-1 {
				buf.WriteString("," + nl)
			}
		}
		buf.WriteString(nl + id + "]")
	case reflect.Map:
		buf.WriteString("{\n")
		for i, k := range v.MapKeys() {
			buf.WriteString(strings.Repeat(" ", indent+2))
			buf.WriteString(k.String() + ": ")
			strVal(v.MapIndex(k), indent+2, buf)
			if i < v.Len()-1 {
				buf.WriteString(",\n")
			}
		}
		buf.WriteString("\n" + strings.Repeat(" ", indent) + "}")
	default:
		format := "%v"
		switch v.Interface().(type) {
		case string:
			format = "%q"
		}
		fmt.Fprintf(buf, format, v.Interface())
	}
}

func main() {
	lambda.Start(HandleRequest)
}
