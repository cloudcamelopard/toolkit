package toolkit

import (
	"bytes"
	"net/http"
	"image"
	"image/png"
	"fmt"
	"os"
	"net/http/httptest"
	"sync"
	"mime/multipart"
	"io"
	"testing"
)

func TestTools_RandomString(t *testing.T) {
	var testTools Tools
	s := testTools.RandomString(10)
	if len(s) != 10 {
		t.Error("Wrong length random string returned")
	}
}

var uploadTests = []struct {
	name string
	allowedTypes []string
	renameFile bool
	errorExpected bool
} {
	{name: "allowed no rename", allowedTypes: []string{"image/jpeg","image/png"}, renameFile:false, errorExpected:false},
	{name: "allowed rename", allowedTypes: []string{"image/jpeg","image/png"}, renameFile:true, errorExpected:false},
	{name: "not allowed", allowedTypes: []string{"image/jpeg"}, renameFile:false, errorExpected:true},
}

func TestTools_UploadFiles(t *testing.T) {
	for _, e := range uploadTests {
		// Set up pipe to avoid buffering
		pr, pw := io.Pipe()
		writer := multipart.NewWriter(pw)
		wg := sync.WaitGroup {}
		wg.Add(1)

		go func() {
			defer writer.Close()
			defer wg.Done()

			/// create the form data field 'file'
			part, err := writer.CreateFormFile("file", "./testdata/img.png")
			if err != nil {
				t.Error(err)
			}

			f, err := os.Open("./testdata/img.png")
			if err != nil {
				t.Error(err)
			}
			defer f.Close()

			img, _, err := image.Decode(f)
			if err != nil {
				t.Error("error decoding image", err)
			}

			err = png.Encode(part, img)
			if err != nil {
				t.Error(err)
			}
		}()

		// read from the pipe which receives data
		request := httptest.NewRequest("POST", "/", pr)
		request.Header.Add("Content-Type", writer.FormDataContentType())

		var testTools Tools 
		testTools.AllowedFileTypes = e.allowedTypes

		uploadedFiles, err := testTools.UploadFiles(request, "./testdata/uploads/", e.renameFile)
		if err != nil && !e.errorExpected {
			t.Error(err)
		}

		if !e.errorExpected {
			if _,err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName)); os.IsNotExist(err) {
				t.Errorf("%s: expected file to exist: %s", e.name, err.Error())
			}

			_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFiles[0].NewFileName))
		}

		if !e.errorExpected && err != nil {
			t.Errorf("%s: error expected but none received", e.name)
		}

		wg.Wait()
	}
}

func TestTools_UploadOneFile(t *testing.T) {
	// Set up pipe to avoid buffering
	pr, pw := io.Pipe()
	writer := multipart.NewWriter(pw)

	go func() {
		defer writer.Close()

		/// create the form data field 'file'
		part, err := writer.CreateFormFile("file", "./testdata/img.png")
		if err != nil {
			t.Error(err)
		}

		f, err := os.Open("./testdata/img.png")
		if err != nil {
			t.Error(err)
		}
		defer f.Close()

		img, _, err := image.Decode(f)
		if err != nil {
			t.Error("error decoding image", err)
		}

		err = png.Encode(part, img)
		if err != nil {
			t.Error(err)
		}
	}()

	// read from the pipe which receives data
	request := httptest.NewRequest("POST", "/", pr)
	request.Header.Add("Content-Type", writer.FormDataContentType())

	var testTools Tools

	uploadedFile, err := testTools.UploadOneFile(request, "./testdata/uploads/", true)
	if err != nil {
		t.Error(err)
	}

	if _,err := os.Stat(fmt.Sprintf("./testdata/uploads/%s", uploadedFile.NewFileName)); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", err.Error())
	}

	_ = os.Remove(fmt.Sprintf("./testdata/uploads/%s", uploadedFile.NewFileName))	
}

func TestTools_CreateDirIfNotExist(t *testing.T) {
	var testTools Tools

	err := testTools.CreateDirIfNotExist("./testdata/myDir")
	if err != nil {
		t.Error(err)
	}

	err = testTools.CreateDirIfNotExist("./testdata/myDir")
	if err != nil {
		t.Error(err)
	}

	_ = os.Remove("./testdata/myDir")
}

var slugTests = []struct {
	name string
	s string
	expected string
	errorExpected bool
} {
	{name: "valid string", s: "now is the time", expected: "now-is-the-time", errorExpected: false},
	{name: "invalid string", s: "@@@@", expected: "", errorExpected: true},
	{name: "empty string", s: "", expected: "", errorExpected: true},
	{name: "complex string", s: "Now is the time for GOOD men of all types!", expected: "now-is-the-time-for-good-men-of-all-types", errorExpected: false},
}

func TestTools_Slugify(t *testing.T) {
	var testTool Tools

	for _, e := range slugTests {
		slug, err := testTool.Slugify(e.s)
		if err != nil && !e.errorExpected {
			t.Errorf("%s: error received when none expected: %s", e.name, err.Error())
		}
		if !e.errorExpected && slug != e.expected {
			t.Errorf("%s: wrong slug returned: expected %s but got %s", e.name, e.expected, slug)
		}
	}
}

func TestTools_DownloadStaticFile(t *testing.T) {
	rr := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	var testTool Tools
	
	testTool.DownloadStaticFile(rr, req, "./testdata", "pic.jpg", "puppy.jpg")

	res := rr.Result()
	defer res.Body.Close()

	if res.Header["Content-Length"][0] != "98827" {
		t.Error("wrong content length of ", res.Header["Content-Length"][0] )
	}

	if res.Header["Content-Disposition"][0] != "attachment; filename=\"puppy.jpg\"" {
		t.Error("wrong content disposition")
	}

	_, err := io.ReadAll(res.Body)
	if err != nil {
		t.Error(err)
	}
}

var jsonTests = []struct {
	name 			string
	json 			string
	errorExpected 	bool
	maxSize 		int
	allowUnknown 	bool
} {
	{ name: "good json", json: `{"foo":"bar"}`, errorExpected: false, maxSize: 1024, allowUnknown: false },
	{ name: "badly formatted json", json: `{"foo":}`, errorExpected: true, maxSize: 1024, allowUnknown: false },
	{ name: "incorrect type", json: `{"foo":33}`, errorExpected: true, maxSize: 1024, allowUnknown: false },
	{ name: "two json files", json: `{"foo":"bar"}{"alpha":"beta"}`, errorExpected: true, maxSize: 1024, allowUnknown: false },
	{ name: "empty body", json: ``, errorExpected: true, maxSize: 1024, allowUnknown: false },
	{ name: "syntax error in json", json: `{"foo": 1"`, errorExpected: true, maxSize: 1024, allowUnknown: false },
	{ name: "unknown field", json: `{"fooo":"bar"}`, errorExpected: true, maxSize: 1024, allowUnknown: false },
	{ name: "allow unknown fields in json", json: `{"fooo":"bar"}`, errorExpected: false, maxSize: 1024, allowUnknown: true },
	{ name: "missing field name", json: `{jack:"bar"}`, errorExpected: true, maxSize: 1024, allowUnknown: true },
	{ name: "file too large", json: `{"foo":"bar"}`, errorExpected: true, maxSize: 4, allowUnknown: false },
	{ name: "not json", json: `hello world!`, errorExpected: true, maxSize: 1024, allowUnknown: false },
}

func TestTools_ReadJSON(t *testing.T) {
	var testTool Tools
	
	for _, e := range jsonTests {
		// set the max file size
		testTool.MaxJSONSize = e.maxSize

		// allow/disallow unknown fields
		testTool.AllowUnknownFields = e.allowUnknown

		// declare a variable to read the decoded JSON into
		var decodedJSON struct {
			Foo string `json:"foo"`
		}

		req, err := http.NewRequest("POST", "/", bytes.NewReader([]byte(e.json)))
		if err != nil {
			t.Log("Error:", err)
		}
		defer req.Body.Close()

		// create a recorder
		rr := httptest.NewRecorder()

		err = testTool.ReadJSON(rr, req, &decodedJSON)

		if e.errorExpected && err == nil {
			t.Errorf("%s: error expected, but none received", e.name)
		}
		if !e.errorExpected && err != nil {
			t.Errorf("%s: error not expected, but one received: %s", e.name, err.Error())
		}
	}
}

func TestTools_WriteJSON(t *testing.T) {
	var testTools Tools

	rr := httptest.NewRecorder()
	payload := JSONResponse {
		Error: false,
		Message: "foo",
	}

	headers := make(http.Header)
	headers.Add("FOO", "BAR")

	err := testTools.WriteJSON(rr, http.StatusOK, payload, headers)
	if err != nil {
		t.Errorf("failed to write JSON: %v", err)
	}
}

