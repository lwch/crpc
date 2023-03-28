package codec

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestRequest(t *testing.T) {
	c := New()
	req, err := http.NewRequest(http.MethodGet, "http://localhost/ping", nil)
	if err != nil {
		t.Fatal(err)
	}
	data, err := c.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}

	var newReq http.Request
	_, err = c.Unmarshal(data, &newReq)
	if err != nil {
		t.Fatal(err)
	}
	if req.Method != newReq.Method {
		t.Fatal("invalid method")
	}
	if req.URL.Path != newReq.URL.Path {
		t.Fatal("invalid url")
	}

	var newReq2 any
	_, err = c.Unmarshal(data, &newReq2)
	if err != nil {
		t.Fatal(err)
	}
	newReq3, ok := newReq2.(*http.Request)
	if !ok {
		t.Fatal("invalid type")
	}
	if req.Method != newReq3.Method {
		t.Fatal("invalid method")
	}
	if req.URL.Path != newReq3.URL.Path {
		t.Fatal("invalid url")
	}
}

func TestResponse(t *testing.T) {
	c := New()
	rep := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("pong")),
	}
	data, err := c.Marshal(rep)
	if err != nil {
		t.Fatal(err)
	}

	var newRep http.Response
	_, err = c.Unmarshal(data, &newRep)
	if err != nil {
		t.Fatal(err)
	}
	if rep.StatusCode != newRep.StatusCode {
		t.Fatal("invalid status code")
	}
	if newRep.Body == nil {
		t.Fatal("invalid body")
	}
	var buf strings.Builder
	if _, err := io.Copy(&buf, newRep.Body); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "pong" {
		t.Fatal("invalid body")
	}

	var newRep2 any
	_, err = c.Unmarshal(data, &newRep2)
	if err != nil {
		t.Fatal(err)
	}
	newRep3, ok := newRep2.(*http.Response)
	if !ok {
		t.Fatal("invalid type")
	}
	if rep.StatusCode != newRep3.StatusCode {
		t.Fatal("invalid status code")
	}
	if newRep3.Body == nil {
		t.Fatal("invalid body")
	}
	var buf2 strings.Builder
	if _, err := io.Copy(&buf2, newRep3.Body); err != nil {
		t.Fatal(err)
	}
	if buf2.String() != "pong" {
		t.Fatal("invalid body")
	}
}

func TestBytes(t *testing.T) {
	c := New()
	data := []byte("codec")
	buf, err := c.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}

	var newBuf []byte
	_, err = c.Unmarshal(buf, &newBuf)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(newBuf) {
		t.Fatal("invalid bytes")
	}

	var newBuf2 any
	_, err = c.Unmarshal(buf, &newBuf2)
	if err != nil {
		t.Fatal(err)
	}
	newBuf3, ok := newBuf2.([]byte)
	if !ok {
		t.Fatal("invalid type")
	}
	if string(data) != string(newBuf3) {
		t.Fatal("invalid bytes")
	}
}
