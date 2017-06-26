package fileupload

import (
	"bytes"
	"io"
	"mime/multipart"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/teambition/gear"
	"net/http/httptest"
)

func multiPartFrom(n int) (io.Reader, string) {
	buf := &bytes.Buffer{}

	mw := multipart.NewWriter(buf)

	mw.WriteField("a", "A")
	mw.WriteField("b", "B")
	mw.WriteField("C", "C")
	mw.WriteField("D", "d")

	mw.WriteField("Abc", "Cba")
	mw.WriteField("d", "t")
	mw.WriteField("e", "55")

	fw, _ := mw.CreateFormFile("testfile", "aa.txt")
	fw.Write([]byte("asdfadsfasdfasdfaefwefaef"))

	if n == 1 {
		f1w, _ := mw.CreateFormFile("file1", "1.txt")
		f1w.Write([]byte("AAABBBCCC1"))
		f2w, _ := mw.CreateFormFile("file2", "2.txt")
		f2w.Write([]byte("AAABBBCCC2"))
		f31w, _ := mw.CreateFormFile("file3", "3.txt")
		f31w.Write([]byte("AAABBBCCC31"))
		f32w, _ := mw.CreateFormFile("file3", "4.txt")
		f32w.Write([]byte("AAABBBCCC32"))
	}

	mw.Close()

	return buf, mw.Boundary()
}

type aWriter struct {
	host     string
	filename string
	content  string
}

func (w *aWriter) Write(ctx *gear.Context, file *FileHeader) error {
	w.host = ctx.Host
	w.filename = file.Filename
	buf := bytes.Buffer{}
	_, err := io.Copy(&buf, file.Reader)
	if err != nil {
		if err.Error() == "http: request body too large" {
			return ErrBodyTooLarge
		}
		return err
	}
	w.content = buf.String()
	return nil
}

type aBodyTemplate struct {
	W   *aWriter `file:"testfile"`
	ABC string   `form:"Abc"`
	D   bool     `form:"d"`
	E   int      `form:"e"`
	A   string   `form:"a"`
	B   string   `form:"b"`
}

func (b *aBodyTemplate) Validate() error {
	if b.ABC == "" {
		return gear.ErrBadRequest.WithMsg("invalid id or pass")
	}
	return nil
}

func TestSaveFileTo(t *testing.T) {
	t.Skip("need operate file system")
	t.Run("", func(t *testing.T) {
		name1, err := saveFileTo(&FileHeader{
			Filename: "3.txt",
			Header:   make(map[string][]string),
			Reader:   bytes.NewReader([]byte("AAABBBCCC31")),
		}, "1.txt")
		if err != nil {
			t.Fatal(err)
		}
		t.Log(name1)

		name2, err := saveFileTo(&FileHeader{
			Filename: "1.txt",
			Header:   make(map[string][]string),
			Reader:   bytes.NewReader([]byte("AAABBBCCC1")),
		}, "")
		if err != nil {
			t.Fatal(err)
		}
		t.Log(name2)
	})
}

func TestWriterCase(t *testing.T) {
	a := assert.New(t)

	fn := writerCase(0, reflect.TypeOf(&aBodyTemplate{}).Elem().Field(0))

	body1 := &aBodyTemplate{W: &aWriter{}}
	rbody1 := reflect.ValueOf(body1).Elem()
	body2 := &aBodyTemplate{W: &aWriter{}}
	rbody2 := reflect.ValueOf(body2).Elem()

	err := fn(rbody1, &gear.Context{Host: "11"},
		&FileHeader{Filename: "a", Reader: bytes.NewReader([]byte("aaa"))})
	if !a.NoError(err) {
		a.FailNow("")
	}
	a.Equal("11", body1.W.host)
	a.Equal("a", body1.W.filename)
	a.Equal("aaa", body1.W.content)

	err = fn(rbody2, &gear.Context{Host: "22"},
		&FileHeader{Filename: "b", Reader: bytes.NewReader([]byte("bbb"))})
	if !a.NoError(err) {
		a.FailNow("")
	}
	a.Equal("22", body2.W.host)
	a.Equal("b", body2.W.filename)
	a.Equal("bbb", body2.W.content)
}

func TestReadMultiPart(t *testing.T) {
	a := assert.New(t)

	newBody := func() *aBodyTemplate {
		return &aBodyTemplate{W: &aWriter{}}
	}

	writers := map[string]handleFunc{
		"testfile": writerCase(0, reflect.TypeOf(newBody()).Elem().Field(0)),
	}

	body1 := newBody()
	r, boundary := multiPartFrom(0)
	mr := multipart.NewReader(r, boundary)
	err := readMultiPart(mr, body1, &gear.Context{Host: "11"}, writers, "form")
	if !a.NoError(err) {
		a.FailNow("")
	}
	a.Equal("11", body1.W.host)
	a.Equal("A", body1.A)
	a.Equal("B", body1.B)
	a.Equal("aa.txt", body1.W.filename)
	a.Equal("asdfadsfasdfasdfaefwefaef", body1.W.content)

	body2 := newBody()
	r, boundary = multiPartFrom(0)
	mr2 := multipart.NewReader(r, boundary)
	err = readMultiPart(mr2, body2, &gear.Context{Host: "22"}, writers, "form")
	if !a.NoError(err) {
		a.FailNow("")
	}
	a.Equal("22", body2.W.host)
}

func TestNew(t *testing.T) {
	t.Run("", func(t *testing.T) {
		a := assert.New(t)
		app := gear.New()

		mw, err := New(func() gear.BodyTemplate {
			return &aBodyTemplate{W: &aWriter{}}
		}, aBodyTemplate{}, 1<<20, "file", "form")
		if !a.NoError(err) {
			a.FailNow("")
		}

		app.Use(mw)

		t.Run("", func(t *testing.T) {
			a := assert.New(t)
			r, boundary := multiPartFrom(0)
			req := httptest.NewRequest("PUT", "/", r)
			req.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)

			res := httptest.NewRecorder()
			ctx := gear.NewContext(app, res, req)

			err = mw(ctx)
			if !a.NoError(err) {
				a.FailNow("")
			}

			body, err := ctx.Any(aBodyTemplate{})
			if !a.NoError(err) {
				a.FailNow("")
			}

			a.Equal("aa.txt", body.(*aBodyTemplate).W.filename)
		})
	})
	t.Run("ErrRequestEntityTooLarge", func(t *testing.T) {
		a := assert.New(t)
		app := gear.New()

		mw, err := New(func() gear.BodyTemplate {
			return &aBodyTemplate{W: &aWriter{}}
		}, aBodyTemplate{}, 1000, "file", "form")
		if !a.NoError(err) {
			a.FailNow("")
		}

		app.Use(mw)

		r, boundary := multiPartFrom(0)

		req := httptest.NewRequest("PUT", "/", r)
		req.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)

		res := httptest.NewRecorder()
		ctx := gear.NewContext(app, res, req)

		err = mw(ctx)

		a.Equal(413, err.(*gear.Error).Code, err.Error())

	})
}
