package verificat

import (
	"bufio"
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"io"
	"log/slog"
	"strings"
)

/*
var (
	awsEndpoint string // Used by customResolver for LocalStack
	awsRegion   string // Used by customResolver for LocalStack
)
*/

// S3ClientAPI is our local interface used to perform specific S3 tasks
// For testing these methods will return customized outputs.
// TODO: ClientData should use this interface instead of *s3.Client
type S3ClientAPI interface {
	GetObject(ctx context.Context,
		params *s3.GetObjectInput,
		optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	ListObjectsV2(ctx context.Context,
		params *s3.ListObjectsV2Input,
		optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

// S3Config returns a configured s3 client,
// currently only one parameter for 'region'
func S3Config(r string) (*s3.Client, error) {
	sdkCfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		slog.Error("Error loading AWS SDK", slog.Any("Error", err))
	}

	s3c := s3.NewFromConfig(sdkCfg, func(o *s3.Options) {
		o.Region = r
		o.UsePathStyle = true
	})

	return s3c, nil
}

// DabS3 interface. Perform operations on S3 with a provided configuration.
type DabS3 interface {
	List() ([]string, error)
	Search() (string, error)
	Get() (string, error)
	Filter() (string, error)
}

// ClientData implements UtilS3
type ClientData struct {
	region, bucket, key, filter string
	client                      S3ClientAPI
}

// NewClientData constructor returns a pointer to a struct literal
// that has been populated with passed-in data used to create an S3 client.
func NewClientData(r, b, k, f string, c S3ClientAPI) *ClientData {
	return &ClientData{
		region: r,
		bucket: b,
		key:    k,
		filter: f,
		client: c,
	}
}

// List provides the full object list of a bucket
func (cd *ClientData) List() ([]string, error) {
	bucketlist, err := cd.client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String(cd.bucket),
	})
	if err != nil {
		slog.Error("failed to list objects in bucket", cd.bucket, cd.key)
		return nil, err
	}

	return bucketToList(bucketlist)
}

// bucketToList translates the S3 type into a slice of strings
func bucketToList(bl *s3.ListObjectsV2Output) ([]string, error) {
	var listable []string
	for _, object := range bl.Contents {
		listable = append(listable, *object.Key)
	}

	return listable, nil
}

// Search identifies whether an object exists in a bucket
// TODO: Use data and/or call ClientData.List()
func (cd *ClientData) Search() (string, error) {
	var found string

	bucketlist, err := cd.List()
	if err != nil {
		slog.Error("failed to list objects in bucket", cd.bucket, cd.key)
		return "", err
	}

	for _, object := range bucketlist {
		if object == cd.key {
			slog.Info("Object Found", slog.String("Key", object))
			found = object
			break
		}
	}

	return found, nil
}

// Get pulls an object from S3.
func (cd *ClientData) Get() (*s3.GetObjectOutput, error) {
	// Get the object from S3
	s3object, err := cd.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: &cd.bucket,
		Key:    &cd.key,
	})
	if err != nil {
		slog.Error("failed to get object", cd.bucket, cd.key)
	}
	// defer s3object.Body.Close()

	return s3object, err
}

// Filter reads object data in S3, then reads it line-by-line.
// If the value of cd.filter is located, the filter passes the line back.
func (cd *ClientData) Filter() (string, error) {
	var filtered string
	var err error

	// Get the object from S3
	s3object, err := cd.Get()
	if err != nil {
		slog.Error("failed to get object", cd.bucket, cd.key)
	}
	// defer s3object.Body.Close()
	defer func() {
		err := s3object.Body.Close()
		if err != nil {
			slog.Error("Request Body failed to Close", slog.Any("Error", err))
			return
		}
	}()

	bodyBytes, err := io.ReadAll(s3object.Body)
	if err != nil {
		slog.Error("failed to read object", cd.bucket, cd.key)
	}
	bodyString := string(bodyBytes)

	scanner := bufio.NewScanner(strings.NewReader(bodyString))
	for scanner.Scan() {
		line := scanner.Text()
		filter, err := NewFilter(cd.filter, line)
		if err != nil {
			slog.Error("failed to parse filter", cd.filter, cd.key)
		}

		if filter == "" {
			slog.Error("failed to filter object", cd.bucket, cd.key)
			break
		} else {
			filtered = strings.Replace(filter, "\n", "", -1)
		}
	}

	return filtered, err
}

// NewFilter takes a string and a line and
// returns the line if it contains the string.
// If the line does not contain the string,
// an error is thrown.
func NewFilter(f, l string) (string, error) {
	if strings.Contains(l, f) {
		return l, nil
	}
	return "", errors.New("filter did not pass")
}

// ResultsS3 contains the bucket found, filter if present, and object list
type ResultsS3 struct {
	Bucket, Filter string
	List           []string
}

// RunS3 is the hub for operations,
// it takes a DabS3 interface to run S3 methods,
// and a string for an object to locate (if any).
func RunS3(i DabS3) (*ResultsS3, error) {
	list, err := i.List()
	if err != nil {
		slog.Error("Listing Failure", slog.Any("Error", err))
		return nil, err
	}

	found, err := i.Search()
	if err != nil {
		slog.Error("Search Failure", slog.Any("Error", err))
		return nil, err
	}

	filtered, err := i.Filter()
	if err != nil {
		slog.Error("Filter Failure", slog.Any("Error", err))
		return nil, err
	}

	return &ResultsS3{Bucket: found, List: list, Filter: filtered}, nil
}
