package shell

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path"

	files "github.com/ipfs/go-ipfs-files"
)

type object struct {
	Hash string
}

func OnlyHash(enabled bool) func(*RequestBuilder) *RequestBuilder {
	return func(rb *RequestBuilder) *RequestBuilder {
		return rb.Option("only-hash", enabled)
	}
}

func Pin(enabled bool) func(*RequestBuilder) *RequestBuilder {
	return func(rb *RequestBuilder) *RequestBuilder {
		return rb.Option("pin", enabled)
	}
}

func Progress(enabled bool) func(*RequestBuilder) *RequestBuilder {
	return func(rb *RequestBuilder) *RequestBuilder {
		return rb.Option("progress", enabled)
	}
}

func RawLeaves(enabled bool) func(*RequestBuilder) *RequestBuilder {
	return func(rb *RequestBuilder) *RequestBuilder {
		return rb.Option("raw-leaves", enabled)
	}
}

func (s *Shell) Add(r io.Reader, options ...func(*RequestBuilder) *RequestBuilder) (string, error) {
	var rc io.ReadCloser
	if rclose, ok := r.(io.ReadCloser); ok {
		rc = rclose
	} else {
		rc = ioutil.NopCloser(r)
	}

	fr := files.NewReaderFile("", "", rc, nil)
	slf := files.NewSliceFile("", "", []files.File{fr})
	fileReader := files.NewMultiFileReader(slf, true)

	var out object
	rb := s.Request("add")
	for _, option := range options {
		rb = option(rb)
	}
	return out.Hash, rb.Body(fileReader).Exec(context.Background(), &out)
}

func (s *Shell) AddLink(target string) (string, error) {
	link := files.NewLinkFile("", "", target, nil)
	slf := files.NewSliceFile("", "", []files.File{link})
	reader := files.NewMultiFileReader(slf, true)

	var out object
	return out.Hash, s.Request("add").Body(reader).Exec(context.Background(), &out)
}

// AddDir adds a directory recursively with all of the files under it
func (s *Shell) AddDir(dir string) (string, error) {
	stat, err := os.Lstat(dir)
	if err != nil {
		return "", err
	}

	sf, err := files.NewSerialFile(path.Base(dir), dir, false, stat)
	if err != nil {
		return "", err
	}
	slf := files.NewSliceFile("", dir, []files.File{sf})
	reader := files.NewMultiFileReader(slf, true)

	resp, err := s.Request("add").
		Option("recursive", true).
		Body(reader).
		Send(context.Background())

	if err != nil {
		return "", nil
	}

	defer resp.Close()

	if resp.Error != nil {
		return "", resp.Error
	}

	dec := json.NewDecoder(resp.Output)
	var final string
	for {
		var out object
		err = dec.Decode(&out)
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
		final = out.Hash
	}

	if final == "" {
		return "", errors.New("no results received")
	}

	return final, nil
}
