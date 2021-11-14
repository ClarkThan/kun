// Code generated by kok; DO NOT EDIT.
// github.com/RussellLuo/kok

package fileupload

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/RussellLuo/kok/pkg/httpcodec"
)

type HTTPClient struct {
	codecs     httpcodec.Codecs
	httpClient *http.Client
	scheme     string
	host       string
	pathPrefix string
}

func NewHTTPClient(codecs httpcodec.Codecs, httpClient *http.Client, baseURL string) (*HTTPClient, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	return &HTTPClient{
		codecs:     codecs,
		httpClient: httpClient,
		scheme:     u.Scheme,
		host:       u.Host,
		pathPrefix: strings.TrimSuffix(u.Path, "/"),
	}, nil
}

func (c *HTTPClient) Upload(ctx context.Context, file *httpcodec.FormFile) (err error) {
	codec := c.codecs.EncodeDecoder("Upload")

	path := "/upload"
	u := &url.URL{
		Scheme: c.scheme,
		Host:   c.host,
		Path:   c.pathPrefix + path,
	}

	reqBody := struct {
		File *httpcodec.FormFile `json:"file"`
	}{
		File: file,
	}
	reqBodyReader, headers, err := codec.EncodeRequestBody(&reqBody)
	if err != nil {
		return err
	}

	_req, err := http.NewRequest("POST", u.String(), reqBodyReader)
	if err != nil {
		return err
	}

	for k, v := range headers {
		_req.Header.Set(k, v)
	}

	_resp, err := c.httpClient.Do(_req)
	if err != nil {
		return err
	}
	defer _resp.Body.Close()

	if _resp.StatusCode < http.StatusOK || _resp.StatusCode > http.StatusNoContent {
		var respErr error
		err := codec.DecodeFailureResponse(_resp.Body, &respErr)
		if err == nil {
			err = respErr
		}
		return err
	}

	return nil
}
