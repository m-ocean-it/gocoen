package analyzer

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"regexp"
	"slices"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

type ConstructorFact struct {
	Entries []ConstructorFactEntry
}

func (cf ConstructorFact) ConstructorNamesString() string {
	constructorNames := make([]string, 0, len(cf.Entries))
	for _, e := range cf.Entries {
		constructorNames = append(constructorNames, `"`+e.ConstructorName+`"`)
	}

	return strings.Join(constructorNames, ", ")
}

type ConstructorFactEntry struct {
	ConstructorName                string
	ConstructorPos, ConstructorEnd token.Pos
}

func (f *ConstructorFact) AFact() {}
func (f *ConstructorFact) String() string {
	constructorNames := make([]string, 0, len(f.Entries))
	for _, e := range f.Entries {
		constructorNames = append(constructorNames, e.ConstructorName)
	}

	return "constructors are " + f.ConstructorNamesString()
}

func NewAnalyzer() *analysis.Analyzer {
	l := newLinter()

	//nolint:exhaustruct
	a := &analysis.Analyzer{
		Name:      "gocoen",
		Doc:       "TODO", // TODO(mmotyshen):
		Run:       l.run,
		Requires:  []*analysis.Analyzer{inspect.Analyzer},
		FactTypes: []analysis.Fact{(*ConstructorFact)(nil)},
	}

	// a.Flags.Init("gocoen", flag.ExitOnError) // TODO(mmotyshen): remove?

	return a
}

type initEntry struct {
	pos token.Pos
	typ *types.TypeName
}

type linter struct{}

func newLinter() *linter {
	l := &linter{}

	return l
}

func (l *linter) run(pass *analysis.Pass) (any, error) {
	insp, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok {
		return nil, errors.New("unexpectedly type is not *inspector.Inspector")
	}

	nodeFilter := []ast.Node{
		(*ast.GenDecl)(nil),
		(*ast.CallExpr)(nil),
		(*ast.AssignStmt)(nil),
		(*ast.CompositeLit)(nil),
		(*ast.ValueSpec)(nil),
	}

	inits := []initEntry{}

	for node := range insp.PreorderSeq(nodeFilter...) {
		l.processNode(pass, node, &inits)
	}

	for _, initEntry := range inits {
		var f ConstructorFact
		ok := pass.ImportObjectFact(initEntry.typ, &f)
		if !ok {
			continue
		}

		var initedWithinOneOfConstructors bool
		for _, factEntry := range f.Entries {
			if initEntry.pos >= factEntry.ConstructorPos && initEntry.pos < factEntry.ConstructorEnd {
				initedWithinOneOfConstructors = true

				break
			}
		}
		if initedWithinOneOfConstructors {
			continue
		}

		pass.Report(analysis.Diagnostic{
			Message: fmt.Sprintf(`"%s" must be constructed with one of these constructors: %s`,
				initEntry.typ.Name(),
				f.ConstructorNamesString(),
			),
			Pos: initEntry.pos,
		})
	}

	return nil, nil //nolint:nilnil
}

func (l *linter) processNode(pass *analysis.Pass, node ast.Node, inits *[]initEntry) {
	switch n := node.(type) {
	case *ast.GenDecl:
		l.processGenDecl(pass, n)
	case *ast.CallExpr:
		l.processCallExpr(pass, n, inits)
	case *ast.CompositeLit:
		l.processCompositeLit(pass, n, inits)
	case *ast.ValueSpec:
		l.processValueSpec(pass, n, inits)
	}
}

func (l *linter) processGenDecl(pass *analysis.Pass, genDecl *ast.GenDecl) {
	if genDecl.Tok != token.TYPE {
		return
	}
	if len(genDecl.Specs) == 0 {
		return
	}
	if genDecl.Doc == nil || len(genDecl.Doc.List) == 0 {
		return
	}

	var constructorNames []string
	for _, commentLine := range genDecl.Doc.List {
		constructorNames = constructorNamesFromDocLine(commentLine.Text)
		if len(constructorNames) > 0 {
			break
		}
	}
	if len(constructorNames) == 0 {
		return
	}

	if len(genDecl.Specs) == 0 {
		pass.Report(analysis.Diagnostic{
			Pos:     genDecl.Pos(),
			End:     genDecl.End(),
			Message: "No type specs are present",
		})

		return
	}

	if len(genDecl.Specs) > 1 {
		pass.Report(analysis.Diagnostic{
			Pos:     genDecl.Pos(),
			End:     genDecl.End(),
			Message: "Multiple specs are not supported",
		})

		return
	}

	typeSpec, ok := genDecl.Specs[0].(*ast.TypeSpec)
	if !ok {
		pass.Report(analysis.Diagnostic{
			Pos:     genDecl.Pos(),
			End:     genDecl.End(),
			Message: "Must be a type spec",
		})

		return
	}

	typeSpecType := pass.TypesInfo.TypeOf(typeSpec.Name)
	if typeSpecType == nil {
		return
	}
	typeSpecObj := pass.TypesInfo.ObjectOf(typeSpec.Name)
	if typeSpecObj == nil {
		return
	}

	entries := make([]ConstructorFactEntry, 0, len(constructorNames))

	for _, cName := range constructorNames {
		constructorObject := pass.Pkg.Scope().Lookup(cName)
		if constructorObject == nil {
			pass.Report(analysis.Diagnostic{
				Pos:     genDecl.Pos(),
				End:     genDecl.End(),
				Message: fmt.Sprintf("Constructor %q does not exist in the same package", cName),
			})

			return
		}
		constructorFunction, ok := constructorObject.(*types.Func)
		if !ok {
			pass.Report(analysis.Diagnostic{
				Pos:     genDecl.Pos(),
				End:     genDecl.End(),
				Message: fmt.Sprintf("Constructor %q must be a function", cName),
			})

			return
		}
		constructorPos := constructorFunction.Pos()
		fnScope := constructorFunction.Scope()
		if fnScope == nil {
			pass.Report(analysis.Diagnostic{
				Pos:     genDecl.Pos(),
				End:     genDecl.End(),
				Message: fmt.Sprintf("Constructor %q is invalid", cName),
			})

			return
		}
		constructorEnd := fnScope.End()

		constructorReturnVars := constructorFunction.Signature().Results()
		var returnTypes []types.Type
		for v := range constructorReturnVars.Variables() {
			returnTypes = append(returnTypes, v.Type())
		}
		if len(returnTypes) == 0 {
			pass.Report(analysis.Diagnostic{
				Pos:     genDecl.Pos(),
				End:     genDecl.End(),
				Message: fmt.Sprintf("Constructor %q does not return anything", cName),
			})

			return
		}

		if !slices.ContainsFunc(
			returnTypes,
			func(rt types.Type) bool {
				return types.Identical(typeSpecType, rt) || types.Identical(types.NewPointer(typeSpecType), rt)
			},
		) {
			pass.Report(analysis.Diagnostic{
				Pos:     genDecl.Pos(),
				End:     genDecl.End(),
				Message: fmt.Sprintf("Constructor %q does not return the corresponding type", cName),
			})

			return
		}

		entries = append(entries, ConstructorFactEntry{
			ConstructorName: cName,
			ConstructorPos:  constructorPos,
			ConstructorEnd:  constructorEnd,
		})

		continue

	}

	if len(entries) > 0 {
		pass.ExportObjectFact(typeSpecObj, &ConstructorFact{Entries: entries})
	}
}

func (l *linter) processCallExpr(pass *analysis.Pass, callExpr *ast.CallExpr, inits *[]initEntry) {
	fn, ok := callExpr.Fun.(*ast.Ident)
	if !ok {
		return
	}

	if fn.Name != "new" {
		return
	}

	if len(callExpr.Args) != 1 {
		return
	}

	ident := typeIdent(callExpr.Args[0])
	if ident == nil {
		return
	}

	typeObj := pass.TypesInfo.TypeOf(ident)
	if typeObj == nil {
		return
	}

	named, ok := typeObj.(*types.Named)
	if !ok || named == nil {
		return
	}

	namedTypeObj := named.Obj()
	if namedTypeObj == nil {
		return
	}

	*inits = append(*inits, initEntry{
		pos: callExpr.Pos(),
		typ: namedTypeObj,
	})
}

func (l *linter) processCompositeLit(pass *analysis.Pass, compositeLit *ast.CompositeLit, inits *[]initEntry) {
	ident := typeIdent(compositeLit.Type)
	if ident == nil {
		return
	}

	typ := pass.TypesInfo.TypeOf(compositeLit)
	if typ == nil {
		return
	}

	named, ok := typ.(*types.Named)
	if !ok || named == nil {
		return
	}

	namedTypeObj := named.Obj()
	if namedTypeObj == nil {
		return
	}

	*inits = append(*inits, initEntry{
		pos: compositeLit.Pos(),
		typ: namedTypeObj,
	})
}

func (l *linter) processValueSpec(pass *analysis.Pass, valueSpec *ast.ValueSpec, inits *[]initEntry) {
	switch t := valueSpec.Type.(type) {
	case *ast.Ident:
		typeObj := pass.TypesInfo.TypeOf(t)
		if typeObj == nil {
			return
		}

		named, ok := typeObj.(*types.Named)
		if !ok || named == nil {
			return
		}

		namedTypeObj := named.Obj()
		if namedTypeObj == nil {
			return
		}

		*inits = append(*inits, initEntry{
			pos: valueSpec.Pos(),
			typ: namedTypeObj,
		})
	case *ast.StarExpr:
		ident := typeIdent(t.X)
		if ident == nil {
			return
		}

		typeObj := pass.TypesInfo.TypeOf(ident)
		if typeObj == nil {
			return
		}

		named, ok := typeObj.(*types.Named)
		if !ok || named == nil {
			return
		}

		namedTypeObj := named.Obj()
		if namedTypeObj == nil {
			return
		}

		*inits = append(*inits, initEntry{
			pos: valueSpec.Pos(),
			typ: namedTypeObj,
		})
	}
}

var directiveRegex = regexp.MustCompile(`#constructor\[([^\]\r\n]+)\]`)

func constructorNamesFromDocLine(docLine string) []string {
	docLine = strings.TrimPrefix(docLine, "// ")
	docLine = strings.TrimSpace(docLine)
	if docLine == "" {
		return nil
	}

	m := directiveRegex.FindStringSubmatch(docLine)
	if len(m) != 2 {
		return nil
	}

	var constructorNames []string
	for cn := range strings.SplitSeq(m[1], ",") {
		constructorNames = append(constructorNames, strings.TrimSpace(cn))
	}

	return constructorNames
}

func typeIdent(expr ast.Expr) *ast.Ident {
	switch id := expr.(type) {
	case *ast.Ident:
		return id
	case *ast.SelectorExpr:
		return id.Sel
	}
	return nil
}
