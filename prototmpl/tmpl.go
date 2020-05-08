package prototmpl

import (
	"encoding/json"
	"fmt"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"regexp"
	"strconv"
)

var substitutionExpressionRegex *regexp.Regexp

func init() {
	var err error
	substitutionExpressionRegex, err = regexp.Compile("\\{\\{(.*)}}")
	if err != nil {
		panic(err)
	}
}

type Template struct {
	subs map[string]substitution
	partialMessage proto.Message
}

func (t* Template) Evaluate(args map[string]interface{}) (proto.Message, error) {
	cloned := proto.Clone(t.partialMessage)

	for name, s := range t.subs {
		walker := protoreflect.ValueOf(cloned.ProtoReflect())
		for _, p := range s.path[:len(s.path)-1] {
			switch {
			case p.field != nil:
				msg := walker.Message()
				//msg, ok := walker.Interface().(protoreflect.Message)
				//if !ok {
				//	return nil, fmt.Errorf("path traversal mismatch: attempted field access but current node is not a message")
				//}

				walker = msg.Get(p.field)
			case p.index != nil:
				// TODO(snowp): Seems like there's no way to do this without panicing on type mismatch?
				l := walker.List()

				walker = l.Get(*p.index)
			}
		}

		last := s.path[len(s.path)-1]

		switch  {
		case last.index != nil:
			// This shouldn't be a Set, we want to insert
			walker.List().Set(*last.index, protoreflect.ValueOf(args[name]))
		case last.field != nil:
			//msg, ok := walker.Interface().(proto.Message)
			msg := walker.Message()
			//if !ok {
			//	return nil, fmt.Errorf("path traversal mismatch: attempted field access but current node is not a message")
			//}

			msg.Set(last.field, protoreflect.ValueOf(args[name]))
		}
	}

	return cloned, nil
}

func templatePathsRecurse(tree interface{}, paths *map[string][]string, currentPath []string) (bool, error) {
	// The only thing we care about finding is string elements, so we don't have to handle all the cases here.
	// We only care about aggregate types and the literal string type: nothing else matters because it would never
	// result in us finding another string.
	switch element := tree.(type) {
	case map[string]interface{}:
		for key, value := range element {
			found, err := templatePathsRecurse(value, paths, append(currentPath, key))
			if err != nil {
				return false, err
			}
			if found {
				delete(element, key)
			}
		}
	case []interface{}:
		var newEntries []interface{}
		for index, value := range element {
			found, err := templatePathsRecurse(value, paths, append(currentPath, strconv.Itoa(index)))
			if err != nil {
				return false, err
			}
			if !found {
				newEntries = append(newEntries, value)
			}
		}

		element = newEntries
	case string:
		result := substitutionExpressionRegex.FindStringSubmatch(element)
		if len(result) > 0 {
			(*paths)[result[1]] = currentPath
		}

		return true, nil
	}

	return false, nil
}

func templatePaths(js string) (map[string][]string, string, error) {
	structured := map[string]interface{}{}
	err := json.Unmarshal([]byte(js), &structured)
	if err != nil {
		return nil, "", err
	}

	paths := map[string][]string{}

	_, err = templatePathsRecurse(structured, &paths, []string{})
	if err != nil {
		return nil, "", err
	}

	marshalled, err := json.Marshal(structured)
	if err != nil {
		return nil, "", err
	}

	return paths, string(marshalled), nil
}

type fieldOrArrayAccess struct {
	index *int
	field protoreflect.FieldDescriptor
}

type substitution struct {
	path []fieldOrArrayAccess
	// Should be the same as path[-1].Kind(), here for convenience
	kind protoreflect.Kind
}

type TemplateCompiler struct {

}

func NewTemplateCompiler() TemplateCompiler {
	return TemplateCompiler{
	}
}

func (tc *TemplateCompiler) createSubstitutions(jsonPaths map[string][]string, messageDescriptor protoreflect.MessageDescriptor) (map[string]substitution, error) {
	// Lots of optimizations that can be done here to avoid traversing the same path multiple times, but for now we do it the good old boring way.
	subs := map[string]substitution{}

	for name, value := range jsonPaths {
		currentDescriptor := messageDescriptor
		var lastFieldDescriptor protoreflect.FieldDescriptor
		var protoPath []fieldOrArrayAccess
		for _, segment := range value {
			// segment is either a numeric value (indicating an array access) or a string (indicating a field access).
			numeric, err := strconv.Atoi(segment)
			if err == nil {
				protoPath = append(protoPath, fieldOrArrayAccess{index: &numeric})
				continue
			}

			if currentDescriptor == nil {
				return nil, fmt.Errorf("attempting to traverse non-message type: next path segment '%s', full path '%s'", segment, value)
			}

			found := false
			for i := 0; i < currentDescriptor.Fields().Len(); i++ {
				f := currentDescriptor.Fields().Get(i)
				if f.JSONName() == segment {
					protoPath = append(protoPath, fieldOrArrayAccess{field: f})
					currentDescriptor = f.Message()
					lastFieldDescriptor = f
					found = true
					break
				}
			}


			if !found {
				return nil, fmt.Errorf("field %s not found in type %s", segment, currentDescriptor.Name())
			}
		}

		subs[name] = substitution{protoPath, lastFieldDescriptor.Kind()}
	}

	return subs, nil
}

func (tc *TemplateCompiler) CompileTemplate(message proto.Message, js string) (*Template, error) {
	paths, prunedJson, err := templatePaths(js)
	if err != nil {
		return nil, err
	}

	subs, err := tc.createSubstitutions(paths, message.ProtoReflect().Descriptor())

	cloned := proto.Clone(message)
	err = protojson.Unmarshal([]byte(prunedJson), cloned)
	if err != nil {
		return nil, err
	}

	return &Template{subs: subs, partialMessage: cloned}, nil
}
