// Package docs Code generated by swaggo/swag. DO NOT EDIT
package docs

import "github.com/swaggo/swag"

const docTemplate = `{
    "schemes": {{ marshal .Schemes }},
    "swagger": "2.0",
    "info": {
        "description": "{{escape .Description}}",
        "title": "{{.Title}}",
        "contact": {},
        "version": "{{.Version}}"
    },
    "host": "{{.Host}}",
    "basePath": "{{.BasePath}}",
    "paths": {
        "/autocomplete": {
            "get": {
                "description": "Returns 'google-like' autocomplete results when the user types in the search box",
                "produces": [
                    "application/json"
                ],
                "summary": "Autocomplete",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Search query",
                        "name": "search",
                        "in": "query",
                        "required": true
                    }
                ],
                "responses": {}
            }
        },
        "/fulltextsearch": {
            "get": {
                "description": "Performs a full-text search on the dashboard",
                "produces": [
                    "application/json"
                ],
                "summary": "Full Text Search",
                "parameters": [
                    {
                        "type": "string",
                        "description": "Search query",
                        "name": "search",
                        "in": "query",
                        "required": true
                    },
                    {
                        "type": "integer",
                        "description": "Page number",
                        "name": "page",
                        "in": "query"
                    },
                    {
                        "type": "integer",
                        "description": "Results limit per page",
                        "name": "limit",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "Field to sort by",
                        "name": "sortBy",
                        "in": "query"
                    },
                    {
                        "type": "string",
                        "description": "Sorting order (asc/desc)",
                        "name": "orderBy",
                        "in": "query"
                    }
                ],
                "responses": {}
            }
        },
        "/searchhistory": {
            "get": {
                "description": "Retrieves the search history of the dashboard",
                "produces": [
                    "application/json"
                ],
                "summary": "Get Dashboard Search History",
                "responses": {}
            },
            "post": {
                "description": "Updates the search history of the dashboard",
                "consumes": [
                    "application/json"
                ],
                "produces": [
                    "application/json"
                ],
                "summary": "Update Dashboard Search History",
                "parameters": [
                    {
                        "description": "Search query",
                        "name": "search",
                        "in": "body",
                        "required": true,
                        "schema": {
                            "type": "string"
                        }
                    }
                ],
                "responses": {}
            }
        }
    }
}`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "",
	Host:             "",
	BasePath:         "",
	Schemes:          []string{},
	Title:            "",
	Description:      "",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
