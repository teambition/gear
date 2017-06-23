package multipart

import (
	"bytes"
	"io"
	"mime/multipart"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/teambition/gear"
)

func MultipartForm() (io.Reader, string) {
	buf := &bytes.Buffer{}
	mw := multipart.NewWriter(buf)
	mw.WriteField("Abc", "Cba")
	mw.WriteField("d", "t")
	mw.WriteField("e", "55")

	f1w, _ := mw.CreateFormFile("file1", "1.txt")
	f1w.Write([]byte("AAABBBCCC1"))
	f2w, _ := mw.CreateFormFile("file2", "2.txt")
	f2w.Write([]byte("AAABBBCCC2"))
	f31w, _ := mw.CreateFormFile("file3", "3.txt")
	f31w.Write([]byte("AAABBBCCC31"))
	f32w, _ := mw.CreateFormFile("file3", "4.txt")
	f32w.Write([]byte("AAABBBCCC32"))

	mw.Close()

	return buf, mw.Boundary()
}

type multipartBodyTemplate struct {
	ABC  string                  `form:"Abc"`
	D    bool                    `form:"d"`
	E    int                     `form:"e"`
	One  *multipart.FileHeader   `file:"file1"`
	All  []*multipart.FileHeader `file:"file3"`
	All2 []*multipart.FileHeader `file:"file2"`
}

func (b *multipartBodyTemplate) Validate() error {
	if b.ABC == "" {
		return gear.ErrBadRequest.WithMsg("invalid id or pass")
	}
	return nil
}

func TestGearFormToStruct(t *testing.T) {
	blob, boundary := MultipartForm()
	mr := multipart.NewReader(blob, boundary)
	data, _ := mr.ReadForm(1 << 20)

	t.Run("Should work", func(t *testing.T) {
		assert := assert.New(t)
		body := &multipartBodyTemplate{}

		if err := FormToStruct(data, body, "form", "file"); err != nil {
			t.Fatal(err)
		}
		assert.Equal("Cba", body.ABC)
		assert.Equal(true, body.D)
		assert.Equal(int(55), body.E)

		assert.Equal(2, len(body.All))
		assert.Equal("3.txt", body.All[0].Filename)
		assert.Equal("4.txt", body.All[1].Filename)

		assert.Equal(1, len(body.All2))
		assert.Equal("2.txt", body.All2[0].Filename)
		assert.Equal("1.txt", body.One.Filename)
	})
}

func TestSaveFileTo(t *testing.T) {
	t.Skip("need operate file system")
	t.Run("tmpfile exist", func(t *testing.T) {
		blob, boundary := MultipartForm()
		mr := multipart.NewReader(blob, boundary)
		form, _ := mr.ReadForm(0)

		name1, err := SaveFileTo(form.File["file3"][0], "1.txt")
		if err != nil {
			t.Fatal(err)
		}
		t.Log(name1)

		name2, err := SaveFileTo(form.File["file1"][0], "")
		if err != nil {
			t.Fatal(err)
		}
		t.Log(name2)
	})
	t.Run("tmpfile not exist", func(t *testing.T) {
		blob, boundary := MultipartForm()
		mr := multipart.NewReader(blob, boundary)
		form, _ := mr.ReadForm(1 << 10)

		name1, err := SaveFileTo(form.File["file3"][0], "1.txt")
		if err != nil {
			t.Fatal(err)
		}
		t.Log(name1)

		name2, err := SaveFileTo(form.File["file1"][0], "")
		if err != nil {
			t.Fatal(err)
		}
		t.Log(name2)
	})
}
