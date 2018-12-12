package happyapi

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi3"
)

const (
	defaultMethod = "GET"
)

type Block struct {
	Number int    `json:"number,omitempty"`
	Hex    string `json:"hex"`
	a      string
}

type Header struct {
	Number int `json:"Number"`
	b      int
}

type APIT struct{}

func (h APIT) BlockHeaderAsString(a Block, b Header, ptr *Block) string {
	return "STRINGIFIED=" + a.a + " ptr:" + ptr.a
}

func (h APIT) BlockToHeader(a Block) Header {
	return Header{42, 69}
}

// Implement Swagger Interface

func (api *APIT) InitSwagger() *openapi2.Swagger {
	return &openapi2.Swagger{
		Host: "localhost",
	}
}

func (api *APIT) IODefaultMethod() string {
	return defaultMethod
}

func (api *APIT) IODefaultPath(methodName string) string {
	return "happy/" + methodName
}

// NOTE: the registry could alternatively use strings (Type Names) as keys for loosey goosey legibility
// also NOTE: it could also alternatively not be a map, eg a slice. the map is nice for being able to lookup examples, but only for that really so far
func (api *APIT) IOParamsRegistry() map[reflect.Type]interface{} {
	return map[reflect.Type]interface{}{
		reflect.TypeOf(Block{}):        Block{42, "0xdeadbeef", "three"},
		reflect.TypeOf(new(Block)):     &Block{42, "0xdeadbeefPOINTER", "zeitgeist"},
		reflect.TypeOf(Header{}):       Header{8, 5},
		reflect.TypeOf(reflect.String): "stringey",
	}
}

func (api *APIT) IOMethodsRegistry() map[string]*MethodReg {
	return map[string]*MethodReg{
		"BlockHeaderAsString": &MethodReg{"POST", "postBHaS"},
		// leave BlockToHeader nil on purpose
	}
}

// Testing
func logSwag(note string, t *testing.T, s *openapi2.Swagger) {
	b, err := json.MarshalIndent(s, "", "    ")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("[swagger.%s] => %s", note, string(b))
}

func TestInitBareSwagger(t *testing.T) {
	s := &openapi2.Swagger{
		Info: openapi3.Info{
			Title:       "test swag",
			Description: "a test of happy swagging",
		},
		Schemes:  []string{"http"},
		Host:     "localhost",
		BasePath: "etc",
	}
	logSwag("initfull", t, s)
	s = defaultOrIncomingSwagger(nil)
	logSwag("initbare", t, s)
}

func TestSwag(t *testing.T) {
	swag, err := Swagger(&APIT{})
	if err != nil {
		t.Fatal(err)
	}
	logSwag("full", t, swag)

}
