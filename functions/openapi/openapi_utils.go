package openapi_functions

func GetAllOperationsJSONPath() string {
	return "$.paths[*]['get','put','post','delete','options','head','patch','trace']"
}
