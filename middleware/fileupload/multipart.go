package fileupload

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"reflect"

	"github.com/teambition/gear"
)

func saveFileTo(file *FileHeader, moveTo string) (_ string, err error) {
	if moveTo != "" {
		moveTo, err = filepath.Abs(moveTo)
		if err != nil {
			return "", err
		}
	}

	var df *os.File
	if moveTo == "" {
		df, err = ioutil.TempFile("", "fileupload-")
		moveTo = df.Name()
	} else {
		df, err = os.Create(moveTo)
	}
	if err != nil {
		return "", err
	}

	_, err = io.Copy(df, file.Reader)
	df.Close()
	if err != nil {
		os.Remove(df.Name())
		return "", err
	}
	return moveTo, nil
}

type FileHeader struct {
	Filename string
	Header   textproto.MIMEHeader
	Reader   io.Reader
}

type Writer interface {
	Write(ctx *gear.Context, file *FileHeader) error
}

type handleFunc func(body reflect.Value, ctx *gear.Context, file *FileHeader) error

var stringType = reflect.TypeOf("")

func stringCase(i int) handleFunc {
	return func(body reflect.Value, ctx *gear.Context, file *FileHeader) error {
		field := body.Field(i)
		filename, err := saveFileTo(file, field.String())
		if err != nil {
			return err
		}
		field.SetString(filename)
		return nil
	}
}

var writerType = reflect.TypeOf((*Writer)(nil)).Elem()

func writerCase(i int, field reflect.StructField) handleFunc {
	if !field.Type.Implements(writerType) {
		panic(field.Name + " not implements fileupload.Writer")
	}
	m, _ := field.Type.MethodByName("Write")
	methodIndex := m.Index
	return func(body reflect.Value, ctx *gear.Context, file *FileHeader) error {
		fn := body.Field(i).Method(methodIndex)
		e := fn.Call([]reflect.Value{reflect.ValueOf(ctx), reflect.ValueOf(file)})
		if e[0].Interface() == nil {
			return nil
		}
		return e[0].Interface().(error)
	}
}

var ErrBodyTooLarge = errors.New("fileupload: request body too large")

func readMultiPart(r *multipart.Reader, body gear.BodyTemplate, ctx *gear.Context, writers map[string]handleFunc, formTag string) error {
	rBody := reflect.ValueOf(body).Elem()

	form := make(map[string][]string)

	maxValueBytes := int64(10 << 20)
	for {
		p, err := r.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			if err.Error() == "multipart: NextPart: http: request body too large" {
				return ErrBodyTooLarge
			}
			//if _, err := r.NextPart(); err.Error() == "multipart: NextPart: http: request body too large" {
			//	return ErrBodyTooLarge
			//}
			return err
		}

		name := p.FormName()
		if name == "" {
			continue
		}
		filename := p.FileName()

		var b bytes.Buffer

		if filename == "" {
			// value, store as string in memory
			n, err := io.CopyN(&b, p, maxValueBytes)
			if err != nil && err != io.EOF {
				return err
			}
			maxValueBytes -= n
			if maxValueBytes == 0 {
				return errors.New("multipart: message too large")
			}
			form[name] = append(form[name], b.String())
			continue
		}

		fn, ok := writers[name]
		if !ok {
			return fmt.Errorf("find a file not allow: %s", name)
		}

		err = fn(rBody, ctx, &FileHeader{
			Filename: filename,
			Header:   p.Header,
			Reader:   p,
		})
		if err != nil {
			return err
		}
	}
	return gear.ValuesToStruct(form, body, formTag)
}

// New creates a file upload middleware to handle multipart form body with file.
//
//	type someWriter struct {
//		host     string
//		filename string
//		content  string
//	}
//	func (w *someWriter) Write(ctx *gear.Context, file *FileHeader) error {
//		w.host = ctx.Host
//		w.filename = file.Filename
//		buf := bytes.Buffer{}
//		_, err := io.Copy(&buf, file.Reader)
//		if err != nil {
//			if err.Error() == "http: request body too large" {
//				return ErrBodyTooLarge
//			}
//			return err
//		}
//		w.content = buf.String()
//		return nil
//	}
//
//	type someBodyTemplate struct {
//		File1 *someWriter `file:"file1"`
//		File2 string `file:"file2"`
//	}
//
//	func (b *someBodyTemplate) Validate() error {
//		return nil
//	}
//
//	newBody := func() gear.BodyTemplate {
//		return &someBodyTemplate{File1: &someWriter{}}
//	}
//
//	mw, err := New(newBody, aBodyTemplate{}, 1<<20, "file", "form")
//	if err != nil {
//	return err
//	}
//
//	app:=gear.New()
//	app.Use(mw)
//
func New(newBody func() gear.BodyTemplate, key interface{}, maxSize int64, fileTag, formTag string) (gear.Middleware, error) {
	bodyType := reflect.TypeOf(newBody())
	if bodyType.Kind() != reflect.Ptr {
		return nil, fmt.Errorf("invalid struct: %v", bodyType)
	}

	bodyType = bodyType.Elem()

	writers := make(map[string]handleFunc)

	for i, n := 0, bodyType.NumField(); i < n; i++ {
		field := bodyType.Field(i)
		tag := field.Tag.Get(fileTag)
		if tag == "" {
			continue
		}
		//todo 检查tag
		switch field.Type {
		case stringType:
			writers[tag] = stringCase(i)
		default:
			writers[tag] = writerCase(i, field)
		}
	}

	return func(ctx *gear.Context) (err error) {
		mediaType := ctx.Get(gear.HeaderContentType)
		mediaType, params, err := mime.ParseMediaType(mediaType)
		if err != nil || mediaType != gear.MIMEMultipartForm {
			return gear.ErrUnsupportedMediaType.WithMsg("unsupported media type")
		}
		boundary, ok := params["boundary"]
		if !ok {
			return http.ErrMissingBoundary
		}

		reader := http.MaxBytesReader(ctx.Res, ctx.Req.Body, maxSize)
		mr := multipart.NewReader(reader, boundary)

		body := newBody()
		err = readMultiPart(mr, body, ctx, writers, formTag)
		if err != nil {
			if err == ErrBodyTooLarge {
				return gear.ErrRequestEntityTooLarge.From(err)
			}
			return gear.ErrBadRequest.From(err)
		}

		err = body.Validate()
		if err != nil {
			return err
		}

		ctx.SetAny(key, body)

		return
	}, nil
}
