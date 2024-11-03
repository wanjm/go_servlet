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

		switch servlet.comment.method {
		case POST, "":
			pathItem.Post = initOperation()
		case GET:
			pathItem.Get = initOperation()
		default:
			fmt.Printf("servlet %s has invalid method %s,which is not supported\n", servlet.Name, servlet.comment.method)
			continue
		}
		paths[strings.Trim(servlet.comment.Url, "\"")] = pathItem
	}
}
func (pkg *Package) addServletToSwagger() {
	if pkg.Project == nil {
		pkg.Project.initSwagger()
	}
	paths := pkg.Project.swag.Paths.Paths
	pkg.FunctionManager.addServletToSwagger(paths)
	for _, class := range pkg.StructMap {
		class.addServletToSwagger(paths)
	}
}
