package main

import (
	"github.com/golang/protobuf/proto"
	plugin_go "github.com/golang/protobuf/protoc-gen-go/plugin"
	"github.com/snowp/prototmpl/prototmpl"
	"io/ioutil"
	"os"
	"path/filepath"
)

func main() {
	b, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(err)
	}

	req := plugin_go.CodeGeneratorRequest{}
	req.ProtoReflect().Descriptor()
	err = proto.Unmarshal(b, &req)
	if err != nil {
		panic(err)
	}

	templateDir := req.GetParameter()
	if len(templateDir) == 0 {
		return
	}

	tc := prototmpl.NewTemplateCompiler()
	var templates []*prototmpl.Template

	filepath.Walk(templateDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}

		b, err := ioutil.ReadAll(f)
		if err != nil {
			return err
		}

		t, err := tc.CompileTemplate(string(b))
		if err != nil {
			return nil
		}

		templates = append(templates, t)
		return nil
	})


}
