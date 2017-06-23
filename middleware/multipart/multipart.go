package multipart

import (
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"os"
	"path/filepath"
	"reflect"

	"github.com/teambition/gear"
	"mime"
	"net/http"
)

var stringType = reflect.TypeOf("")
var fileHeaderType = reflect.TypeOf((*multipart.FileHeader)(nil))
var fileHeaderSliceType = reflect.TypeOf([]*multipart.FileHeader{})

// FormToStruct converts multipart.Form into struct object.
//
//	type multipartBodyTemplate struct {
//		ID     string                  `form:"id"`
//		Pass   string                  `form:"pass"`
//		Photo1 *multipart.FileHeader   `file:"photo1"`
//
//		//if Photo2 is not empty, the file will save to that as a path
//		Photo2 string                  `file:"photo2"`
//		Photo3 []*multipart.FileHeader `file:"photo3"`
//	}
//
//  target := multipartBodyTemplate{}
//
//	FormToStruct(form, &target, "form","file")
func FormToStruct(form *multipart.Form, target interface{}, formTag, fileTag string) (err error) {
	if form == nil {
		return fmt.Errorf("invalid values: <nil>")
	}
	defer form.RemoveAll()

	err = gear.ValuesToStruct(form.Value, target, formTag)
	if err != nil {
		return
	}

	if len(form.File) == 0 {
		return
	}

	rv := reflect.ValueOf(target)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("invalid struct: %v", rv)
	}

	rv = rv.Elem()
	rt := rv.Type()
	n := rv.NumField()
	for i := 0; i < n; i++ {
		fv := rv.Field(i)
		if !fv.CanSet() {
			continue
		}

		fk := rt.Field(i).Tag.Get(fileTag)
		if fk == "" {
			continue
		}

		if fhs, ok := form.File[fk]; ok {
			switch rt.Field(i).Type {
			case stringType:
				name, err := SaveFileTo(fhs[0], fv.String())
				form.File[fk] = fhs[1:]
				if err != nil {
					return err
				}
				fv.SetString(name)
			case fileHeaderType:
				fv.Set(reflect.ValueOf(fhs[0]))
				form.File[fk] = fhs[1:]
			case fileHeaderSliceType:
				fv.Set(reflect.ValueOf(fhs))
				delete(form.File, fk)
			}
		}
	}
	return
}

// SaveFileTo save file to moveTo and return file's abs path,
// if moveTo is empty, save file to temp path.
func SaveFileTo(file *multipart.FileHeader, moveTo string) (string, error) {
	if file == nil {
		return "", fmt.Errorf("invalid values: <nil>")
	}

	var err error
	if moveTo != "" {
		moveTo, err = filepath.Abs(moveTo)
		if err != nil {
			return "", err
		}
	}

	rf := reflect.ValueOf(*file)
	name := rf.FieldByName("tmpfile").String()
	if name != "" {
		if moveTo == "" {
			return name, nil
		}
		err = os.Rename(name, moveTo)
		if err != nil {
			return "", err
		}
		return name, nil
	}
	var df *os.File
	if moveTo == "" {
		df, err = ioutil.TempFile("", "")
		moveTo = df.Name()
	} else {
		df, err = os.Create(moveTo)
	}
	if err != nil {
		return "", err
	}

	sf, err := file.Open()
	if err != nil {
		df.Close()
		return "", err
	}
	_, err = io.Copy(df, sf)
	df.Close()
	sf.Close()
	if err != nil {
		return "", err
	}
	return moveTo, nil
}

type BodyTemplate interface {
	Validate() error
	New() BodyTemplate
}

func New(body BodyTemplate, key interface{}, maxBytes, maxMemory int64) gear.Middleware {
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

		reader := http.MaxBytesReader(ctx.Res, ctx.Req.Body, maxBytes)
		mr := multipart.NewReader(reader, boundary)
		form, err := mr.ReadForm(maxMemory)
		if err != nil {
			if err.Error() == "http: request body too large" {
				return gear.ErrRequestEntityTooLarge.From(err)
			}
			return gear.ErrBadRequest.From(err)
		}

		err = FormToStruct(form, body, "form", "file")
		if err != nil {
			return err
		}

		err = body.Validate()
		if err != nil {
			return err
		}

		ctx.SetAny(key, body)

		return
	}
}
