package codegen

import (
	"fmt"
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
)

// ServiceInfo holds the parsed service metadata.
type ServiceInfo struct {
	Name    string       // from //tool:service annotation
	PkgPath string       // full package import path
	Methods []MethodInfo
}

// MethodInfo holds metadata for a single tool method.
type MethodInfo struct {
	GoName        string            // Go method name (e.g., "GetWeather")
	ToolName      string            // tool name: //tool:name value, or snake_case(GoName)
	Desc          string            // from //tool:desc annotation
	ReqType       types.Type        // the request parameter type (2nd param, after ctx)
	RespType      types.Type        // the first return type
	FieldComments map[string]string // key: "TypeName.FieldName" -> combined doc comment + //tool: annotations
}

// ParseService parses a Go source file and extracts the first //tool:service annotated interface.
// Returns ServiceInfo with all tool methods and their parameter/return types resolved via go/types.
func ParseService(filename string) (*ServiceInfo, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	// Type-check the package using the default importer.
	conf := types.Config{
		Importer: importer.Default(),
	}
	pkg, err := conf.Check(f.Name.Name, fset, []*ast.File{f}, nil)
	if err != nil {
		return nil, fmt.Errorf("type check error: %w", err)
	}

	// Find the //tool:service annotated interface.
	serviceName, typeSpec := findServiceInterface(f)
	if serviceName == "" || typeSpec == nil {
		return nil, fmt.Errorf("no //tool:service annotation found in %s", filename)
	}

	// Look up the type-checked interface.
	ifaceObj := pkg.Scope().Lookup(typeSpec.Name.Name)
	if ifaceObj == nil {
		return nil, fmt.Errorf("interface %s not found in package scope", typeSpec.Name.Name)
	}
	checkedIface, ok := ifaceObj.Type().Underlying().(*types.Interface)
	if !ok {
		return nil, fmt.Errorf("%s is not an interface type", typeSpec.Name.Name)
	}

	// Collect field-level doc comments from all struct types.
	fieldComments := collectFieldComments(f)

	// Process tool methods.
	ifaceAST := typeSpec.Type.(*ast.InterfaceType)
	methods, err := processMethods(ifaceAST, checkedIface, fieldComments)
	if err != nil {
		return nil, err
	}

	return &ServiceInfo{
		Name:    serviceName,
		PkgPath: pkg.Path(),
		Methods: methods,
	}, nil
}

// findServiceInterface walks the AST looking for a type declaration preceded
// by a //tool:service annotation.
func findServiceInterface(f *ast.File) (string, *ast.TypeSpec) {
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE || gd.Doc == nil {
			continue
		}
		if !hasToolAnnotation(gd.Doc, "service") {
			continue
		}
		serviceName := extractToolValue(gd.Doc, "service")
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			if _, ok := ts.Type.(*ast.InterfaceType); ok {
				return serviceName, ts
			}
		}
	}
	return "", nil
}

// processMethods iterates over the interface methods, extracting tool metadata
// from annotations and resolving types via the type-checked interface.
func processMethods(
	ifaceAST *ast.InterfaceType,
	checkedIface *types.Interface,
	fieldComments map[string]string,
) ([]MethodInfo, error) {
	var methods []MethodInfo

	for _, field := range ifaceAST.Methods.List {
		if len(field.Names) == 0 {
			continue // skip embedded interfaces
		}
		methodName := field.Names[0].Name

		if !hasToolAnnotation(field.Doc, "tool") {
			continue // skip methods without //tool:tool
		}

		desc := extractToolValue(field.Doc, "desc")
		toolName := extractToolValue(field.Doc, "name")
		if toolName == "" {
			toolName = SnakeCase(methodName)
		}

		checkedMethod := findExplicitMethod(checkedIface, methodName)
		if checkedMethod == nil {
			return nil, fmt.Errorf("method %s not found in type-checked interface", methodName)
		}

		sig, ok := checkedMethod.Type().(*types.Signature)
		if !ok {
			return nil, fmt.Errorf("method %s has unexpected type %T", methodName, checkedMethod.Type())
		}

		reqType, respType, err := extractRequestResponse(sig)
		if err != nil {
			return nil, fmt.Errorf("method %s: %w", methodName, err)
		}

		methods = append(methods, MethodInfo{
			GoName:        methodName,
			ToolName:      toolName,
			Desc:          desc,
			ReqType:       reqType,
			RespType:      respType,
			FieldComments: fieldComments,
		})
	}

	return methods, nil
}

// findExplicitMethod looks up a method by name in the explicit methods of an interface.
func findExplicitMethod(iface *types.Interface, name string) *types.Func {
	for i := 0; i < iface.NumExplicitMethods(); i++ {
		if m := iface.ExplicitMethod(i); m.Name() == name {
			return m
		}
	}
	return nil
}

// extractRequestResponse validates the method signature and returns the request
// (2nd param) and response (1st return) types. The expected signature is:
//
//	(ctx context.Context, req T) (R, error)
func extractRequestResponse(sig *types.Signature) (types.Type, types.Type, error) {
	if sig.Params().Len() != 2 {
		return nil, nil, fmt.Errorf("expected 2 parameters (ctx, request), got %d", sig.Params().Len())
	}
	if sig.Results().Len() != 2 {
		return nil, nil, fmt.Errorf("expected 2 results (response, error), got %d", sig.Results().Len())
	}

	errType := sig.Results().At(1).Type()
	if !types.Identical(errType, types.Universe.Lookup("error").Type()) {
		return nil, nil, fmt.Errorf("second return value must be error")
	}

	return sig.Params().At(1).Type(), sig.Results().At(0).Type(), nil
}

// collectFieldComments extracts field-level doc comments from all struct types
// in the AST. Key format: "TypeName.FieldName".
func collectFieldComments(f *ast.File) map[string]string {
	comments := make(map[string]string)
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.TYPE {
			continue
		}
		for _, spec := range gd.Specs {
			ts, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}
			st, ok := ts.Type.(*ast.StructType)
			if !ok {
				continue
			}
			typeName := ts.Name.Name
			for _, field := range st.Fields.List {
				for _, name := range field.Names {
					if !name.IsExported() {
						continue
					}
					key := typeName + "." + name.Name
					var parts []string
					if field.Doc != nil {
						parts = append(parts, commentGroupRaw(field.Doc))
					}
					if field.Comment != nil {
						parts = append(parts, commentGroupRaw(field.Comment))
					}
					if len(parts) > 0 {
						comments[key] = strings.TrimSpace(strings.Join(parts, "\n"))
					}
				}
			}
		}
	}
	return comments
}

// commentGroupRaw returns the raw comment text with // or /* */ markers preserved.
func commentGroupRaw(cg *ast.CommentGroup) string {
	if cg == nil {
		return ""
	}
	var parts []string
	for _, c := range cg.List {
		parts = append(parts, strings.TrimSpace(c.Text))
	}
	return strings.Join(parts, "\n")
}

// hasToolAnnotation checks if a comment group contains a specific //tool: directive.
func hasToolAnnotation(doc *ast.CommentGroup, directive string) bool {
	if doc == nil {
		return false
	}
	target := "//tool:" + directive
	for _, c := range doc.List {
		trimmed := strings.TrimSpace(c.Text)
		if trimmed == target || strings.HasPrefix(trimmed, target+" ") {
			return true
		}
	}
	return false
}

// extractToolValue extracts the value from a //tool:<key> <value> line.
func extractToolValue(doc *ast.CommentGroup, key string) string {
	if doc == nil {
		return ""
	}
	prefix := "//tool:" + key + " "
	for _, c := range doc.List {
		trimmed := strings.TrimSpace(c.Text)
		if strings.HasPrefix(trimmed, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, prefix))
		}
	}
	return ""
}
