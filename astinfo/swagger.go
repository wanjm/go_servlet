package astinfo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	if len(project.cfg.SwaggerCfg.UrlPrefix) > 0 {
		if !strings.HasPrefix(project.cfg.SwaggerCfg.UrlPrefix, "/") {
			project.cfg.SwaggerCfg.UrlPrefix = "/" + project.cfg.SwaggerCfg.UrlPrefix
		}
		project.cfg.SwaggerCfg.UrlPrefix = strings.TrimSuffix(project.cfg.SwaggerCfg.UrlPrefix, "/")
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
		var url = strings.Trim(servlet.comment.Url, "\"")
		if len(url) == 0 {
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
		if len(servlet.Results) > 1 {
			field0 := servlet.Results[0]
			if field0.class == nil {
				field0.findStruct(false)
			}
			objRef = swagger.getRefOfStruct(field0.class.(*Struct))
		}
		addSecurity(servlet, operation) //apix中使用了全局的header，暂时不显示
		var response spec.Response = swagger.getSwaggerResponse(objRef)
		operation.Responses.StatusCodeResponses[200] = response
		paths[swagger.project.cfg.SwaggerCfg.UrlPrefix+url] = pathItem
	}
}
func addSecurity(function *Function, operation *spec.Operation) {
	for _, header := range function.comment.security {
		operation.Security = append(operation.Security, map[string][]string{
			header: {"string"},
		})
	}
	for _, s := range function.pkg.Project.servers {
		if s.name == function.comment.serverName {
			for url, filter := range s.urlFilters {
				servletUrl := strings.Trim(filter.comment.Url, "\"")
				filterUrl := strings.Trim(url, "\"")
				if strings.Contains(servletUrl, filterUrl) {
					for _, header := range filter.comment.security {
						operation.Security = append(operation.Security, map[string][]string{
							header: {"string"},
						})
					}
				}
			}
		}
	}
}
func (swagger *Swagger) GenerateCode(cfg *SwaggerCfg) string {
	if cfg.Token == "" {
		return ""
	}
	project := swagger.project
	for name, pkg := range project.Package {
		_ = name
		swagger.addServletFromPackage(pkg)
	}
	swaggerJson, _ := swagger.swag.MarshalJSON()
	cmdMap := map[string]interface{}{
		"input": string(swaggerJson),
		"options": map[string]interface{}{
			"targetEndpointFolderId":        cfg.ServletFolder,
			"targetSchemaFolderId":          cfg.SchemaFolder,
			"endpointOverwriteBehavior":     "OVERWRITE_EXISTING",
			"schemaOverwriteBehavior":       "OVERWRITE_EXISTING",
			"updateFolderOfChangedEndpoint": false,
			"prependBasePath":               false,
		},
	}
	data, _ := json.Marshal(cmdMap)
	url := "https://api.apifox.com/v1/projects/" + cfg.ProjectId + "/import-openapi?locale=zh-CN"
	req, _ := http.NewRequest("POST", url, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Apifox-Api-Version", "2024-03-28")
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	req.Header.Set("User-Agent", "Apifox/1.0.0 (https://apifox.com)")
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("error:%v\n", err)
	}
	content, _ := io.ReadAll(response.Body)
	fmt.Printf("response:%v\n", string(content))
	// fmt.Printf("swagger:%s\n", cmdMap["input"])
	return (string(data))
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
	rawpkg := swagger.project.rawPkg
	/*
		"expireType": { //结构体格式
			"$ref": "#/definitions/schema.ExpireType"
		},
		"expireValue": { //原始类型格式
			"type": "integer"
		},
		"gradeIds": {  //数组格式
			"type": "array",
			"items": {
				"type": "integer"
			}
		},
	*/
	for _, field := range class.fields {
		var name = field.jsonName
		if len(name) == 0 {
			name = firstLower(field.name)
		}
		schema := spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: field.comment,
			},
		}
		typeName := field.typeName
		if len(typeName) > 0 {
			if field.pkg == rawpkg {
				//原始类型
				typeName = getRawTypeString(typeName)
				schema.Type = []string{typeName}
				if typeName == "array" {
					schema.Items = &spec.SchemaOrArray{
						Schema: &spec.Schema{},
					}
				}
			} else {
				//数组格式
				schema.Ref = *swagger.getRefOfStruct(field.class.(*Struct))
			}

		}
		schemas[name] = schema
	}
	ref, _ := spec.NewRef("#/definitions/" + class.Name)
	swagger.definitions[class.Name] = &ref
	swagger.swag.Definitions[class.Name] = spec.Schema{
		SchemaProps: result,
	}
	return &ref
}

func (swagger *Swagger) initResponseResult() {
	rawpkg := swagger.project.rawPkg
	class := Struct{
		Name: "ResponseResult",
		fields: []*Field{
			{
				name:     "code",
				typeName: "int",
				pkg:      rawpkg,
			},
			{
				name:     "msg",
				typeName: "string",
				pkg:      rawpkg,
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
