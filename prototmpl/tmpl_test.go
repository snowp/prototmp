package prototmpl

import (
	"bytes"
	prototmpl_test "github.com/snowp/prototmpl/test/proto/prototmpl"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"testing"
	gtmpl "text/template"
)

const simpleTemplate = `
{
  "foo": "{{baz}}"
}
`

const nestedTemplate = `
{
  "foo": {
    "bar": "{{baz}}",
	"bar2": {
      "key": "{{baz2}}"
    }
  }
}
`

const simpleArrayTemplate = `
{
   "foo": [
     "{{tmpl1}}",
     "{not a template}",
     "{{tmpl2}}"
   ]
}
`

const nestedArrayTemplate = `
{
   "foo": [
     {},
     { "bar": "{{tmpl}}" },
     {}
   ]
}
`

func TestTemplatePaths(t *testing.T) {
	tcs := []struct {
		name            string
		template        string
		expectedEntries map[string][]string
	}{
		{"simple", simpleTemplate, map[string][]string{"baz": {"foo"}}},
		{"nested", nestedTemplate, map[string][]string{"baz": {"foo", "bar"}, "baz2": {"foo", "bar2", "key"}}},
		{"array", simpleArrayTemplate, map[string][]string{"tmpl1": {"foo", "0"}, "tmpl2": {"foo", "2"}}},
		{"nested array", nestedArrayTemplate, map[string][]string{"tmpl": {"foo", "1", "bar"}}},
	}

	for _, tc := range tcs {
		paths, _, err := templatePaths(tc.template)
		assert.Nil(t, err, tc.name)
		assert.Equal(t, tc.expectedEntries, paths, tc.name)
	}
}

const typedTemplate = `
{
	"one": 1,
	"two": "{{two}}"
}
`

func TestSubstitutionGeneration(t *testing.T) {
	tc := NewTemplateCompiler()

	template, err := tc.CompileTemplate(&prototmpl_test.Foo{}, typedTemplate)
	assert.Nil(t, err)

	r, err := template.Evaluate(map[string]interface{}{"two": "templated"})
	assert.Nil(t, err)
	assert.True(t, proto.Equal(r, &prototmpl_test.Foo{
		One: 1,
		Two: "templated",
	}))
}

func BenchmarkNewTemplateCompiler(b *testing.B) {
	tc := NewTemplateCompiler()

	template, err := tc.CompileTemplate(&prototmpl_test.Foo{}, typedTemplate)
	if err != nil {
		panic(err)
	}

	args := map[string]interface{}{"two": "templated"}
	// run the Fib function b.N times
	for n := 0; n < b.N; n++ {
		_, _ = template.Evaluate(args)
	}
}

const goTmpl = `
{
  "one": "1",
  "two": "{{.Two}}"
}
`
func BenchmarkJsonTemplating(b *testing.B) {
	t, err := gtmpl.New("bench").Parse(goTmpl)
	if err != nil {
		b.Fatal(err)
	}

	args := struct {Two string } { "templated"}
	for n := 0; n < b.N; n++ {
		var buffer []byte
		bbuffer := bytes.NewBuffer(buffer)
		_ = t.Execute(bbuffer, args)

		foo := prototmpl_test.Foo{}
		_ = protojson.Unmarshal(bbuffer.Bytes(), &foo)
	}
}

func helperFunction(two string) *prototmpl_test.Foo {
	return &prototmpl_test.Foo{
		One: 1,
		Two: two,
	}
}

func BenchmarkHelperFunction(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_ = helperFunction("templated")
	}
}
