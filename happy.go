package main

import (
	"encoding/json"
	"errors"
	"log"
	"reflect"

	"github.com/alecthomas/jsonschema"
)

var (
	ErrNotRegistered = errors.New("unregistered type")
)

type Swaggerer interface {
	IOExamplesRegistry() map[reflect.Type]interface{}
}

func MakeSwag(sw Swaggerer) ([]byte, error) {
	// get the registry from the interface method
	reg := sw.IOExamplesRegistry()

	apiT := reflect.TypeOf(sw)
	for i := 0; i < apiT.NumMethod(); i++ {
		method := apiT.Method(i)
		// log.Println("[dbug] method=", method, "method.variadic?=", method.Type.IsVariadic())

		if method.Name == "IOExamplesRegistry" {
			continue
		}

		methodNumIn := method.Type.NumIn()
		log.Println("method=", method.Name, "num inputs=", methodNumIn, "variadic?=", method.Type.IsVariadic())
		for j := 0; j < methodNumIn; j++ {
			// get arguments in
			in := method.Type.In(j)

			// TODO: understand me
			if in.Name() == "" {
				continue
			}

			log.Println("INARGS/index:", j,
				"name=",
				in.Name(),
				"in.string=",
				in.String(),
				"in.refval=",
				reflect.New(method.Type),
			)

			// grab the example var and demo the schema for that
			ex, ok := reg[in]
			if !ok {
				return nil, ErrNotRegistered
			}

			v := jsonschema.ReflectFromType(in)
			if v == nil {
				log.Println("v nil")
			} else {
				b, _ := json.MarshalIndent(v, "", "    ")
				exb, _ := json.Marshal(ex)
				log.Printf("def-> %s\n   example-> %s", string(b), string(exb))
			}

		}

		// get arguments out
		methodNumOut := method.Type.NumOut()
		log.Println("method=", method.Name, "num outputs=", methodNumOut)
		for k := 0; k < methodNumOut; k++ {
			out := method.Type.Out(k)
			log.Println("out=", k, "out.name=", out.Name(), "out.string=", out.String())
		}
	}

	return nil, nil
}
