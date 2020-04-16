package codegen

var AllowedPaths = map[string]FileType{
	"/transform/decode/{role_name}":    FileTypeDataSource,
	"/transform/encode/{role_name}":    FileTypeDataSource,
	"/transform/role/{name}":           FileTypeResource,
	"/transform/role":                  FileTypeResource,
	"/transform/transformation/{name}": FileTypeResource,
	"/transform/transformation":        FileTypeResource,
	"/transform/template/{name}":       FileTypeResource,
	"/transform/template":              FileTypeResource,
	"/transform/alphabet/{name}":       FileTypeResource,
	"/transform/alphabet":              FileTypeResource,
}
