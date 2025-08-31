package verificat

import (
	"bytes"
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"io"
	"reflect"
	"testing"
	"time"
)

var (
	region = "us-west-2"
	// Create a fake bucket listing
	mockBucketListing = &s3.ListObjectsV2Output{
		Contents: []types.Object{
			{
				Key:          aws.String("file1.txt"),
				Size:         aws.Int64(1024),
				LastModified: aws.Time(time.Now()),
			},
			{
				Key:          aws.String("file2.txt"),
				Size:         aws.Int64(2048),
				LastModified: aws.Time(time.Now()),
			},
		},
		IsTruncated: aws.Bool(false),
	}
)

// mockBucketObject creates a fake object with data
func mockBucketObject() *s3.GetObjectOutput {
	// mock content
	objectData := []byte("craquemattic\njohncage\nmortonfeldman\n")

	// mock S3 response
	s3Response := &s3.GetObjectOutput{
		Body:          io.NopCloser(bytes.NewReader(objectData)),
		ContentLength: aws.Int64(int64(len(objectData))),
		ContentType:   aws.String("text/plain"),
		ETag:          aws.String("\"mock-etag\""),
		LastModified:  aws.Time(time.Now()),
		Metadata: map[string]string{
			"file1.txt": *aws.String("file1.txt"),
		},
		VersionId: aws.String("mock-version-1"),
	}

	return s3Response
}

func TestS3Config(t *testing.T) {
	sdkCfg, err := config.LoadDefaultConfig(context.TODO())
	assertError(t, err, nil)

	t.Run("S3 clients are equal on the top struct level", func(t *testing.T) {
		// Create a new s3 client with the same configuration as S3Config
		want := s3.NewFromConfig(sdkCfg, func(o *s3.Options) {
			o.Region = region
			o.UsePathStyle = true
		})

		// Create a new s3 client using S3Config
		got, err := S3Config(region)
		assertError(t, err, nil)

		// Confirm that S3Config created the client with a comparable config
		// IgnoreUnexported will disregard unexported (i.e. uncomparable) substructs
		if diff := cmp.Diff(got, want, cmpopts.IgnoreUnexported(s3.Client{})); diff != "" {
			t.Error(diff)
		}
	})
}

// MockS3Client provides fake outputs for configured S3 methods
// It is essentially an S3 Client that only gets responses from these.
type MockS3Client struct {
	ListObjectsV2Output *s3.ListObjectsV2Output
	GetObjectOutput     *s3.GetObjectOutput
}

// ListObjectsV2 on MockS3Client returns its configured output
// i.e. the value set for MockS3Client.ListObjectsV2Output
func (m *MockS3Client) ListObjectsV2(ctx context.Context,
	params *s3.ListObjectsV2Input,
	optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	return m.ListObjectsV2Output, nil
}

// GetObject on MockS3Client returns its configured output
// i.e. the value set for MockS3Client.GetObjectOutput
func (m *MockS3Client) GetObject(ctx context.Context,
	params *s3.GetObjectInput,
	optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return m.GetObjectOutput, nil
}

// TestBucketToList uses these mocks to serve
// a fake bucket with fake contents.
func TestBucketToList(t *testing.T) {
	t.Run("Returns bucket list as a slice of strings", func(t *testing.T) {
		want := []string{"file1.txt", "file2.txt"}
		got, err := bucketToList(mockBucketListing)

		assertMultiString(t, got, want)
		assertNoError(t, err)
	})
}

func TestClientData_List(t *testing.T) {
	// Create mock client
	mockClient := &MockS3Client{
		ListObjectsV2Output: mockBucketListing,
	}

	t.Run("Interface returns bucket list as a slice of strings", func(t *testing.T) {
		mockRun := NewClientData("a", "b", "c", "m", mockClient)

		got, err := mockRun.List()
		want := []string{"file1.txt", "file2.txt"}

		assertNoError(t, err)
		assertMultiString(t, got, want)
	})
}

func TestClientData_Search(t *testing.T) {
	// Create mock client
	mockClient := &MockS3Client{
		ListObjectsV2Output: mockBucketListing,
	}

	t.Run("Interface returns object found", func(t *testing.T) {
		mockRun := NewClientData("a", "b", "file1.txt", "m", mockClient)

		got, err := mockRun.Search()
		want := "file1.txt"

		assertNoError(t, err)
		assertString(t, got, want)
	})
}

func TestClientData_Get(t *testing.T) {
	// Get a mock bucket object
	mockObject := mockBucketObject()
	defer mockObject.Body.Close()

	// Create mock client with the response
	mockClient := &MockS3Client{
		GetObjectOutput: mockObject,
	}

	t.Run("Interface returns object", func(t *testing.T) {
		mockRun := NewClientData("a", "b", "file1.txt", "m", mockClient)

		got, err := mockRun.Get()
		want := mockObject

		assertNoError(t, err)
		if want != got {
			t.Errorf("want: %v, got: %v", want, got)
		}
	})
}

func TestClientData_Filter(t *testing.T) {
	// Get a mock bucket object
	mockObject := mockBucketObject()
	defer mockObject.Body.Close()

	// Create mock client with the response
	mockClient := &MockS3Client{
		GetObjectOutput: mockObject,
	}

	t.Run("Returns filtered line", func(t *testing.T) {
		mockRun := NewClientData("a", "b", "file1.txt", "q", mockClient)

		got, err := mockRun.Filter()
		want := "craquemattic"

		assertNoError(t, err)
		if want != got {
			t.Errorf("want: %v, got: %v", want, got)
		}
	})
}

// NewClientData is a struct constructor function.
// Test that it returns a proper struct with
// variables used for S3 actions and a field for the client.
func TestNewClientData(t *testing.T) {
	testclient, err := S3Config(region)
	assertError(t, err, nil)

	// Literal struct to compare the composed struct
	want := struct {
		region, bucket, key, filter string
		client                      S3ClientAPI
	}{
		region: region,
		bucket: "b",
		key:    "c",
		filter: "m",
		client: testclient,
	}

	t.Run("Returns the correct number of fields", func(t *testing.T) {
		got := *NewClientData(region, "b", "c", "m", testclient)
		gotSize := reflect.TypeOf(got).NumField()
		wantSize := reflect.TypeOf(want).NumField()
		if gotSize != wantSize {
			t.Errorf("Got has %v fields, but wants %v", gotSize, wantSize)
		}
	})
}

// mockCData is identical to ClientData...
type mockClientData struct {
	region, bucket, key, filter string
	client                      s3.Client
}

// List mocks the expected string for use with testing RunS3
func (cd *mockClientData) List() ([]string, error) {
	// Encode an error into the return if /.region/ is set to "z"
	// This will let us check for errors from RunS3
	if cd.region == "z" {
		return nil, errors.New("region error")
	}
	return []string{"c", "d", "e"}, nil
}

// Search mocks the expected string for use with testing RunS3
// For consistency this matches the mockClientData key
func (cd *mockClientData) Search() (string, error) {
	// Encode an error into the return if /.bucket/ is set to "z"
	// This will let us check for errors from RunS3
	if cd.bucket == "z" {
		return "", errors.New("bucket error")
	}
	return "c", nil
}

// Get mock
func (cd *mockClientData) Get() (string, error) {
	panic("implement me")
}

// Filter mocks the successful return of a matched line for testing RunS3
func (cd *mockClientData) Filter() (string, error) {
	// Encode an error into the return if /.filter/ is set to "z"
	// This will let us check for errors from RunS3
	if cd.filter == "z" {
		return "", errors.New("filter error")
	}
	return "m", nil
}

func TestRunS3(t *testing.T) {
	mockRun := &mockClientData{region: "a", bucket: "b", key: "c", filter: "m"}

	t.Run("returns an expected search string", func(t *testing.T) {
		got, _ := RunS3(mockRun)
		want := "c"

		assertString(t, got.Bucket, want)
	})

	t.Run("returns no error", func(t *testing.T) {
		_, err := RunS3(mockRun)

		assertNoError(t, err)
	})

	t.Run("returns an error from List", func(t *testing.T) {
		// The dependency injected here will return an error
		// if the /region/ field is set to the string "z"
		mockRun := &mockClientData{region: "z", bucket: "b", key: "c"}
		_, err := RunS3(mockRun)

		assertHasError(t, err)
	})

	t.Run("returns an error from Search", func(t *testing.T) {
		// The dependency injected here will return an error
		// if the /bucket/ field is set to the string "z"
		mockRun := &mockClientData{region: "a", bucket: "z", key: "c"}
		_, err := RunS3(mockRun)

		assertHasError(t, err)
	})

	t.Run("returns an error from Filter", func(t *testing.T) {
		// The dependency injected here will return an error
		// if the /filter/ field is set to the string "z"
		mockRun := &mockClientData{region: "a", bucket: "b", key: "c", filter: "z"}
		_, err := RunS3(mockRun)

		assertHasError(t, err)
	})

	t.Run("returns a bucket contents listing", func(t *testing.T) {
		got, _ := RunS3(mockRun)
		want := []string{"c", "d", "e"}

		if diff := cmp.Diff(got.List, want); diff != "" {
			t.Error(diff)
			t.Errorf("Got:\n %v \nWant\n %v", got, want)
		}
	})

	t.Run("returns a filtered line", func(t *testing.T) {
		got, _ := RunS3(mockRun)
		want := "m"

		assertString(t, got.Filter, want)
	})
}

func TestNewFilter(t *testing.T) {
	mockRun := &mockClientData{region: "a", bucket: "b", key: "c", filter: "m"}

	// Returns the filterline if the filter value is a substring
	t.Run("returns a filtered line", func(t *testing.T) {
		filterline := "a line with the letter m"
		got, _ := NewFilter(mockRun.filter, filterline)

		assertString(t, got, filterline)
	})

	// Errors when it doesn't find the substring
	t.Run("returns an error without a filtered line", func(t *testing.T) {
		filterline := "a line without the letter before n"
		_, err := NewFilter(mockRun.filter, filterline)

		assertHasError(t, err)
	})
}
