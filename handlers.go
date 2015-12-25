package fasthttp

import (
	"io"
	"fmt"
	"bytes"
	"mime/multipart"
)

func NewContinueHandler(onContinue func(req *Request) bool) RequestHandler {
	return func (ctx *RequestCtx) {
		req:=&ctx.Request
		if !req.Header.noBody() {
			contentLength := req.Header.ContentLength()
			if contentLength > 0 {
				// See http://www.w3.org/Protocols/rfc2616/rfc2616-sec8.html for 100-Continue behavior
				expect := req.Header.Peek("Expect")
				if len(expect) > 0 {
					lowercaseBytes(expect)
					if bytes.Compare(expect, []byte("100-continue")) == 0 {
						if(onContinue!=nil && onContinue(&ctx.Request)){
							ctx.Connection().Write([]byte("HTTP/1.1 100 Continue\r\n\r\n"))
						} else {
							ctx.Connection().Write([]byte("HTTP/1.1 100 Continue\r\n\r\n"))
						}
					} else {
						ctx.Connection().Write([]byte("HTTP/1.1 417 Expectation Failed\r\n\r\n"))
					}
				}
			}
		}
	}
}

func NewBodyParser(maxBodySize int) RequestHandler {
	return func(ctx *RequestCtx) {
		req:=&ctx.Request
		boundary := req.Header.MultipartFormBoundary()
		if len(boundary) == 0 {
				if req.HasBody() {
					contentLength := req.Header.ContentLength()
					var err error
					fmt.Println("Parsing body")
					req.body, err = readBody(ctx.Reader, contentLength, maxBodySize, req.body)
					fmt.Println("Parsing body",string(req.body))
					if err != nil {
						panic(err)
						return
					}
					req.Header.SetContentLength(len(req.body))
				}
		}
	}
}

func readMultipartFormBody(r io.Reader, boundary []byte, maxBodySize, maxInMemoryFileSize int) (*multipart.Form, error) {
	// Do not care about memory allocations here, since they are tiny
	// compared to multipart data (aka multi-MB files) usually sent
	// in multipart/form-data requests.
	if maxBodySize > 0 {
		r = io.LimitReader(r, int64(maxBodySize))
	}
	mr := multipart.NewReader(r, string(boundary))
	f, err := mr.ReadForm(int64(maxInMemoryFileSize))
	if err != nil {
		return nil, fmt.Errorf("cannot read multipart/form-data body: %s", err)
	}
	return f, nil
}

func NewMultipartFormParser(maxBodySize int) RequestHandler {
	return func(ctx *RequestCtx) {
		req:=&ctx.Request
		var err error
		contentLength := req.Header.ContentLength()
		// Pre-read multipart form data of known length.
		// This way we limit memory usage for large file uploads, since their contents
		// is streamed into temporary files if file size exceeds defaultMaxInMemoryFileSize.
		boundary := req.Header.MultipartFormBoundary()
		if len(boundary) > 0 {
			if contentLength > 0 {
				if(maxBodySize!=0 && maxBodySize<contentLength){
					contentLength=maxBodySize
				}
				//We were stalling on waiting for an ending CRLF, RFC2616/4.1 "an HTTP/1.1 client must not preface or follow request with an extra CRLF"
				//which was causing curl to hang waiting on a post request
				fmt.Println("Parsing form")
				req.multipartForm, err = readMultipartFormBody(ctx.Reader, boundary, contentLength, defaultMaxInMemoryFileSize)
				fmt.Println("Done Parsing form",err)
				return
			}
		}
		return
	}
}

