package main

import (
	"os"
	"log"
	"bytes"
	"image"
	"errors"
	"strconv"
	"context"
	"net/http"
	"image/gif"
	"image/png"
	"image/jpeg"
	"image/color"
	"path/filepath"

	"golang.org/x/image/draw"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/s3manager"
)

type StepFunctionsRequestParameter struct {
	Action    string             `json:"action"`
	Key       string             `json:"key"`
	Type      string             `json:"type"`
	Icon      IconParameter      `json:"icon"`
	Thumbnail ThumbnailParameter `json:"thumbnail"`
}

type IconParameter struct {
	Diameter string `json:"diameter"`
	Bgcolor  string `json:"bgcolor"`
}

type ThumbnailParameter struct {
	Width   string `json:"width"`
	Height  string `json:"height"`
	Bgcolor string `json:"bgcolor"`
}

type StepFunctionsResponseParameter struct {
	StatusCode int    `json:"StatusCode"`
	Key        string `json:"Key"`
}

var cfg aws.Config
var s3Client *s3.Client

const layout           string = "2006-01-02 15:04"
const layout2          string = "20060102150405"
const layout3          string = "20060102150405.000"
const bucketResultPath string = "result"

func HandleRequest(ctx context.Context, request StepFunctionsRequestParameter) (StepFunctionsResponseParameter, error) {
	var err error
	if len(request.Action) > 0 {
		switch request.Action {
		case "convert" :
			if imgSrc, _, e := getImage(ctx, request.Key); e == nil {
				err = saveImage(imgSrc, request.Type, request.Key, request.Action)
			} else {
				err = e
			}
		case "icon" :
			if imgSrc, imgType, e := getImage(ctx, request.Key); e == nil {
				col, e := strconv.ParseInt(request.Icon.Bgcolor, 16, 32)
				if e != nil {
					err = e
				} else {
					diameter, e := strconv.Atoi(request.Icon.Diameter)
					if e != nil {
						err = e
					} else {
						err = circleMaskImage(imgSrc, imgSrc.Bounds(), imgType, diameter, uint16(col), request.Key)
					}
				}
			} else {
				err = e
			}
		case "thumbnail" :
			if imgSrc, imgType, e := getImage(ctx, request.Key); e == nil {
				col, e := strconv.ParseInt(request.Thumbnail.Bgcolor, 16, 32)
				if e != nil {
					err = e
				} else {
					width, e := strconv.Atoi(request.Thumbnail.Width)
					if e != nil {
						err = e
					} else {
						height, e := strconv.Atoi(request.Thumbnail.Height)
						if e != nil {
							err = e
						} else {
							err = scaleImage(imgSrc, imgSrc.Bounds(), imgType, width, height, uint16(col), request.Key)
						}
					}
				}
			} else {
				err = e
			}
		}
	}
	if err != nil {
		return StepFunctionsResponseParameter {
			StatusCode: http.StatusInternalServerError,
			Key:        request.Key,
		}, err
	}
	return StepFunctionsResponseParameter {
		StatusCode: http.StatusOK,
		Key:        request.Key,
	}, nil
}

func uploadImage(extension string, filedata []byte, key string) error {
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
		return errors.New("this extension is invalid")
	}
	uploader := s3manager.NewUploader(cfg)
	_, err := uploader.Upload(&s3manager.UploadInput{
		ACL: s3.ObjectCannedACLPublicRead,
		Bucket: aws.String(os.Getenv("BUCKET_NAME")),
		Key: aws.String(key),
		Body: bytes.NewReader(filedata),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		log.Print(err)
		return err
	}
	return nil
}

func getImage(ctx context.Context, key string)(image.Image, string, error) {
	if s3Client == nil {
		s3Client = s3.New(cfg)
	}
	req := s3Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(os.Getenv("BUCKET_NAME")),
		Key:    aws.String(key),
	})
	res, err := req.Send(ctx)
	if err != nil {
		log.Print(err)
		return nil, "", err
	}

	rc := res.GetObjectOutput.Body
	defer rc.Close()
	return image.Decode(rc)
}

func scaleImage(imgSrc image.Image, rctSrc image.Rectangle, imgType string, dstWidth int, dstHeight int, col uint16, key string) error {
	imgDst := image.NewRGBA(image.Rect(0, 0, dstWidth, dstHeight))
	if col != 0 {
		for x := 0; x < dstWidth; x++ {
			for y := 0; y < dstHeight; y++ {
				imgDst.Set(x, y, color.Gray16{col})
			}
		}
	}
	var drawBounds image.Rectangle
	if dstWidth * rctSrc.Dy() > dstHeight * rctSrc.Dx() {
		tmpWidth := (dstWidth - (rctSrc.Dx() * dstHeight / rctSrc.Dy()))/2
		drawBounds = image.Rect(tmpWidth, 0, dstWidth - tmpWidth, dstHeight)
	} else {
		tmpHeight := (dstHeight - (rctSrc.Dy() * dstWidth / rctSrc.Dx()))/2
		drawBounds = image.Rect(0, tmpHeight, dstWidth, dstHeight - tmpHeight)
	}
	draw.BiLinear.Scale(imgDst, drawBounds, imgSrc, rctSrc, draw.Over, nil)

	return saveImage(imgDst, imgType, key, "thumbnail_" + strconv.Itoa(dstWidth) + "_" + strconv.Itoa(dstHeight))
}

func circleMaskImage(imgSrc image.Image, rctSrc image.Rectangle, imgType string, dstDiameter int, col uint16, key string) error {
	p := image.Point{dstDiameter/2, dstDiameter/2}
	r := dstDiameter / 2
	imgDst := image.NewRGBA(image.Rect(0, 0, dstDiameter, dstDiameter))
	if col != 0 {
		for x := 0; x < dstDiameter; x++ {
			for y := 0; y < dstDiameter; y++ {
				imgDst.Set(x, y, color.Gray16{col})
			}
		}
	}
	imgDst_ := image.NewRGBA(image.Rect(0, 0, dstDiameter, dstDiameter))
	var drawBounds image.Rectangle
	if dstDiameter * rctSrc.Dy() > dstDiameter * rctSrc.Dx() {
		tmpHeight := (dstDiameter - (rctSrc.Dy() * dstDiameter / rctSrc.Dx()))/2
		drawBounds = image.Rect(0, tmpHeight, dstDiameter, dstDiameter - tmpHeight)
	} else {
		tmpWidth := (dstDiameter - (rctSrc.Dx() * dstDiameter / rctSrc.Dy()))/2
		drawBounds = image.Rect(tmpWidth, 0, dstDiameter - tmpWidth, dstDiameter)
	}
	draw.BiLinear.Scale(imgDst_, drawBounds, imgSrc, rctSrc, draw.Over, nil)
	imgPointX := (imgDst_.Bounds().Dx() - dstDiameter) / 2
	imgPointY := (imgDst_.Bounds().Dy() - dstDiameter) / 2
	draw.DrawMask(imgDst, imgDst.Bounds(), imgDst_, image.Point{imgPointX,imgPointY}, &circle{p, r}, image.ZP, draw.Over)

	return saveImage(imgDst, imgType, key, "icon_" + strconv.Itoa(dstDiameter))
}

func saveImage(imgSrc image.Image, imgType string, key string, suffix string) error {
	if imgSrc == nil {
		log.Print("Image is nil")
	}
	dst := new(bytes.Buffer)

	if imgType == "jpeg" {
		imgType = "jpg"
	}
	switch imgType {
	case "jpg":
		if err := jpeg.Encode(dst, imgSrc, &jpeg.Options{Quality: 100}); err != nil {
			log.Print(err)
			return err
		}
	case "gif":
		if err := gif.Encode(dst, imgSrc, nil); err != nil {
			log.Print(err)
			return err
		}
	case "png":
		if err := png.Encode(dst, imgSrc); err != nil {
			log.Print(err)
			return err
		}
	default:
		log.Print("Image Format error")
	}
	return uploadImage("." + imgType, dst.Bytes(), createNewKey(key, suffix, imgType))
}

type circle struct {
	p image.Point
	r int
}

func (c *circle) ColorModel() color.Model {
	return color.AlphaModel
}

func (c *circle) Bounds() image.Rectangle {
	return image.Rect(c.p.X-c.r, c.p.Y-c.r, c.p.X+c.r, c.p.Y+c.r)
}

func (c *circle) At(x, y int) color.Color {
	x_, y_, r_ := float64(x-c.p.X)+0.5, float64(y-c.p.Y)+0.5, float64(c.r)
	if x_*x_+y_*y_ < r_*r_ {
		return color.Alpha{255}
	}
	return color.Alpha{0}
}

func createNewKey(key string, suffix string, newExtension string) string {
	if len(suffix) < 1 {
		return key
	}
	extension := filepath.Ext(key)
	return key[:(len(key) - len(extension))] + "_" + suffix + "." + newExtension
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
