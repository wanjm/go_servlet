package astinfo

import (
	"fmt"
	"strings"

	"github.com/go-openapi/spec"
)

type Swagger struct {
	swag           *spec.Swagger
	project        *Project
	definitions    map[string]*spec.Ref
	responseResult *Struct
}

func NewSwagger(project *Project) (result *Swagger) {
	var swag = &spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Swagger: "2.0",
			Info: &spec.Info{
				InfoProps: spec.InfoProps{
					Contact: &spec.ContactInfo{},
					License: nil,
				},
				VendorExtensible: spec.VendorExtensible{
					Extensions: spec.Extensions{},
				},
			},
			Paths: &spec.Paths{
				Paths: make(map[string]spec.PathItem),
				VendorExtensible: spec.VendorExtensible{
					Extensions: nil,
				},
			},
			Definitions:         make(map[string]spec.Schema),
			SecurityDefinitions: make(map[string]*spec.SecurityScheme),
		},
		VendorExtensible: spec.VendorExtensible{
			Extensions: nil,
		},
	}
	result = &Swagger{
		swag:        swag,
		project:     project,
		definitions: make(map[string]*spec.Ref),
	}
	result.initResponseResult()
	return
}

func initOperation() *spec.Operation {
	return &spec.Operation{
		OperationProps: spec.OperationProps{
			ID:           "",
			Description:  "",
			Summary:      "",
			Security:     nil,
			ExternalDocs: nil,
			Deprecated:   false,
			Tags:         []string{},
			Consumes:     []string{},
			Produces:     []string{},
			Schemes:      []string{},
			Parameters:   []spec.Parameter{},
			Responses: &spec.Responses{
				VendorExtensible: spec.VendorExtensible{
					Extensions: spec.Extensions{},
				},
				ResponsesProps: spec.ResponsesProps{
					Default:             nil,
					StatusCodeResponses: make(map[int]spec.Response),
				},
			},
		},
		VendorExtensible: spec.VendorExtensible{
			Extensions: spec.Extensions{},
		},
	}
}
func (swagger *Swagger) addServletFromFunctionManager(pkg *FunctionManager) {
	paths := swagger.swag.Paths.Paths
	for _, servlet := range pkg.servlets {
		if servlet.comment.Url == "" {
			fmt.Printf("servlet %s has no url\n", servlet.Name)
			continue
		}
		pathItem := spec.PathItem{}
		operation := initOperation()
		var parameter []spec.Parameter
		switch servlet.comment.method {
		case POST, "":
			pathItem.Post = operation
			var props spec.SchemaProps
			_ = props
			if len(servlet.Params) > 1 && servlet.Params[1].class != nil {
				ref := swagger.getRefOfStruct(servlet.Params[1].class.(*Struct))
				parameter = append(parameter, spec.Parameter{
					ParamProps: spec.ParamProps{
						Name:     "body",
						In:       "body",
						Required: true,
						Schema: &spec.Schema{
							SchemaProps: spec.SchemaProps{
								Ref: *ref,
							},
						},
					},
				})
			}

		case GET:
			pathItem.Get = operation
		default:
			fmt.Printf("servlet %s has invalid method %s,which is not supported\n", servlet.Name, servlet.comment.method)
			continue
		}
		operation.Parameters = parameter
		var objRef *spec.Ref
		if len(servlet.Results) > 1 && servlet.Results[0].class != nil {
			objRef = swagger.getRefOfStruct(servlet.Results[0].class.(*Struct))
		}
		var response spec.Response = swagger.getSwaggerResponse(objRef)
		operation.Responses.StatusCodeResponses[200] = response
		paths[strings.Trim(servlet.comment.Url, "\"")] = pathItem
	}
}
func (swagger *Swagger) GenerateCode() string {
	project := swagger.project
	for name, pkg := range project.Package {
		_ = name
		swagger.addServletFromPackage(pkg)
	}
	json, _ := swagger.swag.MarshalJSON()
	return (string(json))
}
func (swagger *Swagger) addServletFromPackage(pkg *Package) {
	swagger.addServletFromFunctionManager(&pkg.FunctionManager)
	for _, class := range pkg.StructMap {
		if class.comment.serverType == SERVLET {
			swagger.addServletFromFunctionManager(&class.FunctionManager)
		}
	}
}

func (swagger *Swagger) getRefOfStruct(class *Struct) *spec.Ref {
	if ref, ok := swagger.definitions[class.Name]; ok {
		return ref
	}
	schemas := make(map[string]spec.Schema)
	result := spec.SchemaProps{
		Type:       []string{"object"},
		Properties: schemas,
	}
	for _, field := range class.fields {
		schemas[field.name] = spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type:        []string{field.typeName},
				Description: "zhushi",
			},
		}
	}
	ref, _ := spec.NewRef("#/definitions/" + class.Name)
	swagger.definitions[class.Name] = &ref
	swagger.swag.Definitions[class.Name] = spec.Schema{
		SchemaProps: result,
	}
	return &ref
}

func (swagger *Swagger) initResponseResult() {
	class := Struct{
		Name: "ResponseResult",
		fields: []*Field{
			{
				name:     "code",
				typeName: "int",
			},
			{
				name:     "msg",
				typeName: "string",
			},
			{
				name: "obj",
			},
		},
	}
	swagger.getRefOfStruct(&class)
	swagger.responseResult = &class
}

func (swagger *Swagger) getSwaggerResponse(objRef *spec.Ref) spec.Response {
	respoinseResult := swagger.getRefOfStruct(swagger.responseResult)
	var result = spec.Response{
		ResponseProps: spec.ResponseProps{
			Schema: &spec.Schema{
				SchemaProps: spec.SchemaProps{
					AllOf: []spec.Schema{{
						SchemaProps: spec.SchemaProps{
							Ref: *respoinseResult,
						},
					}},
				},
			},
		},
	}
	if objRef == nil {
		return result
	}
	ref := spec.Schema{
		SchemaProps: spec.SchemaProps{
			Type: []string{"object"},
			Properties: map[string]spec.Schema{
				"obj": {
					SchemaProps: spec.SchemaProps{
						Ref: *objRef,
					},
				},
			},
		},
	}
	result.Schema.AllOf = append(result.Schema.AllOf, ref)
	return result
}
