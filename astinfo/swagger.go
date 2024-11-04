package astinfo

import (
	"fmt"
	"strings"

	"github.com/go-openapi/spec"
)

func (project *Project) initSwagger() {
	project.swag = &spec.Swagger{
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
func (pkg *FunctionManager) addServletToSwagger(paths map[string]spec.PathItem) {
	for _, servlet := range pkg.servlets {
		if servlet.comment.Url == "" {
			fmt.Printf("servlet %s has no url\n", servlet.Name)
			continue
		}
		pathItem := spec.PathItem{}
		operation := initOperation()
		var parameter []spec.Parameter
		var response spec.Response = getSwaggerResponse()
		switch servlet.comment.method {
		case POST, "":
			pathItem.Post = operation
			var props spec.SchemaProps
			_ = props
			if len(servlet.Params) > 1 && servlet.Params[1].class != nil {
				ref, err := spec.NewRef("#/definitions/" + servlet.Params[1].class.(*Struct).Name)
				if err != nil {
					fmt.Printf("servlet %s has invalid class %s\n", servlet.Name, servlet.Params[1].class.(*Struct).Name)
					continue
				}
				parameter = append(parameter, spec.Parameter{
					ParamProps: spec.ParamProps{
						Name:     "body",
						In:       "body",
						Required: true,
						Schema: &spec.Schema{
							SchemaProps: spec.SchemaProps{
								Ref: ref,
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
		operation.Responses.StatusCodeResponses[200] = response
		paths[strings.Trim(servlet.comment.Url, "\"")] = pathItem
	}
}
func (pkg *Package) addServletToSwagger() {
	paths := pkg.Project.swag.Paths.Paths
	pkg.FunctionManager.addServletToSwagger(paths)
	for _, class := range pkg.StructMap {
		class.addServletToSwagger(paths)
	}
}

func (class *Struct) getStructProperties() (result spec.SchemaProps) {
	schemas := make(map[string]spec.Schema)
	result = spec.SchemaProps{
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
	return
}

func getSwaggerResponse() spec.Response {
	respoinseResult, _ := spec.NewRef("#/definitions/ResponseResult")
	var result = spec.Response{
		ResponseProps: spec.ResponseProps{
			Schema: &spec.Schema{
				SchemaProps: spec.SchemaProps{
					AllOf: []spec.Schema{{
						SchemaProps: spec.SchemaProps{
							Ref: respoinseResult,
						},
					}, {
						SchemaProps: spec.SchemaProps{
							Type: []string{"object"},
						},
					}},
				},
			},
		},
	}
	return result
}
