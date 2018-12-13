package happyapi

import (
	"errors"
	"reflect"

	"github.com/getkin/kin-openapi/openapi2"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
)

var (
	ErrNotSupported  = errors.New("unsupported value")
	ErrNotRegistered = errors.New("unregistered value")

	errEmptyType = errors.New("empty type")

	defaultMethods = make(map[Swaggerer]string)
)

var gen *openapi3gen.Generator

type MethodReg struct {
	Method string
	Path   string
}

type Swaggerer interface {
	// IOParamsRegistry eg. types.Block :: types.Block{42, "0xdeadbeef", time.Now()}
	IOParamsRegistry() map[reflect.Type]interface{}

	// IOMethodsRegistry
	// eg. api.GetBlock :: eth/getBlock OR eth_getBlock
	// eg. api.GetBlock :: POST
	IOMethodsRegistry() map[string]*MethodReg
}

func mustSwagger(in *openapi2.Swagger) *openapi2.Swagger {
	if in == nil {
		return &openapi2.Swagger{}
	}
	return in
}

func getParameter(reg map[reflect.Type]interface{}, in reflect.Type) (*openapi2.Parameter, error) {
	// TODO: understand me
	if in.Name() == "" {
		return nil, errEmptyType
	}

	param := &openapi2.Parameter{}

	// grab the example var and demo the schema for that
	_, ok := reg[in]
	if !ok {
		return nil, ErrNotRegistered
	}

	// return an error if a method
	switch in.Kind() {
	case reflect.Func, reflect.Interface:
		return nil, ErrNotSupported
	}

	v, err := gen.GenerateSchemaRef(in)
	if err != nil {
		return param, err
	}

	param.Ref = v.Ref
	param.Name = in.Name()
	param.Schema = v

	param.Description = v.Value.Description
	param.Required = len(v.Value.Required) > 0
	param.UniqueItems = v.Value.UniqueItems
	param.ExclusiveMin = v.Value.ExclusiveMin
	param.ExclusiveMax = v.Value.ExclusiveMax
	param.Type = v.Value.Type
	param.Format = v.Value.Format
	param.Enum = v.Value.Enum

	// // openapi3?
	// param.Min = v.Value.Min
	// param.Max = v.Value.Max

	param.MaxLength = v.Value.MaxLength
	param.MinLength = v.Value.MinLength
	param.Pattern = v.Value.Pattern

	return param, nil
}

func storeSchemaTypeInstance(gen *openapi3gen.Generator, t reflect.Type, r *openapi3.SchemaRef) {
	if gen.Types == nil {
		gen.Types = make(map[reflect.Type]*openapi3.SchemaRef)
	}
	gen.Types[t] = r
}

func saveSchemaDefinition(swag *openapi2.Swagger, gen *openapi3gen.Generator, i interface{}) {
	toi := reflect.TypeOf(i)
	s, err := gen.GenerateSchemaRef(toi)
	if err != nil {
		panic(err.Error())
	}

	swag.Definitions[reflect.TypeOf(i).Name()] = s
}

func getResponse(reg map[reflect.Type]interface{}, out reflect.Type) (string, *openapi2.Response, error) {
	// TODO: understand me
	s := out.Name()
	if s == "" {
		return s, nil, errEmptyType
	}

	res := &openapi2.Response{}

	ex, ok := reg[out]
	if !ok {
		return s, nil, ErrNotRegistered
	}

	// return an error if a method
	switch out.Kind() {
	case reflect.Func, reflect.Interface:
		return s, nil, ErrNotSupported
	}

	v, err := gen.GenerateSchemaRef(out)
	if err != nil {
		return s, res, err
	}

	res.Ref = v.Ref
	res.Description = v.Value.Description
	res.Schema = v
	res.Examples = map[string]interface{}{
		"0": ex,
	}

	return s, res, nil
}

func swaggererOwns(methodName string) bool {
	reservedMethods := []string{}
	r := reflect.TypeOf(struct{ Swaggerer }{})
	nr := r.NumMethod()
	for n := 0; n < nr; n++ {
		m := r.Method(n).Name
		reservedMethods = append(reservedMethods, m)
	}
	isReserved := func(n string) bool {
		for _, v := range reservedMethods {
			if n == v {
				return true
			}
		}
		return false
	}
	return isReserved(methodName) || methodName == "Swagger"
}

// Swagger generates a Swagger OpenAPIv2 scheme.
func Swagger(sw Swaggerer, swag *openapi2.Swagger, service interface{}, defaultMethod func(string) string, defaultPath func(string) string) (*openapi2.Swagger, error) {
	if gen == nil {
		gen = openapi3gen.NewGenerator()
	}

	// TODO: either use a separate funciton for creation of an swagger and then one for appending to an existing one,
	// or establish an Swagger :registered: as a third parameter and check for nil, initing if nil and
	// appending if not.
	// swag := mustSwagger(swagIn)
	if swag.Definitions == nil {
		swag.Definitions = make(map[string]*openapi3.SchemaRef)
	}

	// if service is a slice, iterate the below under the same swag var

	// NOTES: TypeOf(y).IsVariadic
	// TODO: maybe not, see above instead
	serviceK := reflect.TypeOf(service).Kind()
	if serviceK == reflect.Slice {

	}

	paramsReg := sw.IOParamsRegistry()
	methodsReg := sw.IOMethodsRegistry()

	apiT := reflect.TypeOf(service)
	for i := 0; i < apiT.NumMethod(); i++ {

		oper := &openapi2.Operation{}

		method := apiT.Method(i)

		// skip Swaggerer interface's own methods
		if swaggererOwns(method.Name) {
			continue
		}

		methodNumIn := method.Type.NumIn()

		oper.Parameters = openapi2.Parameters{} // init
	PARAMSLOOP:
		for j := 0; j < methodNumIn; j++ {
			// get arguments in
			in := method.Type.In(j)
			p, err := getParameter(paramsReg, in)
			if err == errEmptyType {
				continue PARAMSLOOP
			} else if err == ErrNotRegistered {
				continue PARAMSLOOP
			} else if err != nil {
				return swag, err
			}

			oper.Parameters = append(oper.Parameters, p)
			saveSchemaDefinition(swag, gen, paramsReg[in])
		}

		// get responses out
		methodNumOut := method.Type.NumOut()

		oper.Responses = make(map[string]*openapi2.Response)
	RETURNSLOOP:
		for k := 0; k < methodNumOut; k++ {
			out := method.Type.Out(k)
			s, res, err := getResponse(paramsReg, out)
			if err == errEmptyType {
				continue RETURNSLOOP
			} else if err == ErrNotRegistered {
				continue RETURNSLOOP
			} else if err != nil {
				return swag, err
			}
			if oper.Responses == nil {
				oper.Responses = make(map[string]*openapi2.Response)
			}
			oper.Responses[s] = res

			// TODO handle struct descriptions better
			switch reflect.TypeOf(k).Kind() {
			case reflect.Struct:
				saveSchemaDefinition(swag, gen, paramsReg[out])
			default:
				saveSchemaDefinition(swag, gen, paramsReg[out])
			}
		}

		mr, ok := methodsReg[method.Name]
		if !ok {
			mr = &MethodReg{
				defaultMethod(method.Name),
				// sw.IODefaultMethod(),
				defaultPath(method.Name),
				// sw.IODefaultPath(method.Name),
			}
		}
		swag.AddOperation(mr.Path, mr.Method, oper)
	}
	return swag, nil
}
