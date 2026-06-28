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
	ConstructorName string
	Pos             token.Pos
	End             token.Pos
}

func (f *ConstructorFact) AFact() {}
func (f *ConstructorFact) String() string {
	return "constructor is " + f.ConstructorName
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

type initedEntry struct {
	pos token.Pos
	typ *types.TypeName
}

type data struct {
	zeroValues        []initedEntry
	nilValues         []initedEntry
	compositeLiterals []initedEntry
	typeAliases       []initedEntry
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

	data := data{}

	for node := range insp.PreorderSeq(nodeFilter...) {
		l.processNode(pass, node, &data)
	}

	for _, entry := range data.compositeLiterals {
		var f ConstructorFact
		ok := pass.ImportObjectFact(entry.typ, &f)
		if !ok {
			continue
		}

		// if used inside T's constructor - ignore
		if entry.pos >= f.Pos && entry.pos < f.End {
			continue
		}

		pass.Report(analysis.Diagnostic{
			Message: fmt.Sprintf(`"%s" must be constructed with "%s"`, entry.typ.Name(), f.ConstructorName),
			Pos:     entry.pos,
		})
	}

	for _, entry := range data.zeroValues {
		var f ConstructorFact
		ok := pass.ImportObjectFact(entry.typ, &f)
		if !ok {
			continue
		}

		// if used inside T's constructor - ignore
		if entry.pos >= f.Pos && entry.pos < f.End {
			continue
		}

		pass.Report(analysis.Diagnostic{
			Message: fmt.Sprintf(`"%s" must be constructed with "%s"`, entry.typ.Name(), f.ConstructorName),
			Pos:     entry.pos,
		})
	}

	for _, entry := range data.nilValues {
		var f ConstructorFact
		ok := pass.ImportObjectFact(entry.typ, &f)
		if !ok {
			continue
		}

		// if used inside T's constructor - ignore
		if entry.pos >= f.Pos && entry.pos < f.End {
			continue
		}

		pass.Report(analysis.Diagnostic{
			Message: fmt.Sprintf(`"%s" must be constructed with "%s"`, entry.typ.Name(), f.ConstructorName),
			Pos:     entry.pos,
		})
	}

	return nil, nil //nolint:nilnil
}

func (l *linter) processNode(pass *analysis.Pass, node ast.Node, data *data) {
	switch n := node.(type) {
	case *ast.GenDecl:
		l.processGenDecl(pass, n)
	case *ast.CallExpr:
		l.processCallExpr(pass, n, data)
	case *ast.CompositeLit:
		l.processCompositeLit(pass, n, data)
	case *ast.ValueSpec:
		l.processValueSpec(pass, n, data)
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

	var constructorName string
	for _, commentLine := range genDecl.Doc.List {
		constructorName = constructorNameFromDocLine(commentLine.Text)
		if constructorName != "" {
			break
		}
	}
	if constructorName == "" {
		return
	}

	constructorObject := pass.Pkg.Scope().Lookup(constructorName)
	if constructorObject == nil {
		pass.Report(analysis.Diagnostic{
			Pos:     genDecl.Pos(),
			End:     genDecl.End(),
			Message: fmt.Sprintf("Constructor %q does not exist in the same package", constructorName),
		})

		return
	}
	constructorFunction, ok := constructorObject.(*types.Func)
	if !ok {
		pass.Report(analysis.Diagnostic{
			Pos:     genDecl.Pos(),
			End:     genDecl.End(),
			Message: fmt.Sprintf("Constructor %q must be a function", constructorName),
		})

		return
	}
	constructorPos := constructorFunction.Pos()
	fnScope := constructorFunction.Scope()
	if fnScope == nil {
		pass.Report(analysis.Diagnostic{
			Pos:     genDecl.Pos(),
			End:     genDecl.End(),
			Message: fmt.Sprintf("Constructor %q is invalid", constructorName),
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
			Message: fmt.Sprintf("Constructor %q does not return anything", constructorName),
		})

		return
	}

	for _, s := range genDecl.Specs {
		typeSpec, ok := s.(*ast.TypeSpec)
		if !ok {
			continue
		}

		typeSpecType := pass.TypesInfo.TypeOf(typeSpec.Name)

		if !slices.ContainsFunc(
			returnTypes,
			func(rt types.Type) bool {
				return types.Identical(typeSpecType, rt) || types.Identical(types.NewPointer(typeSpecType), rt)
			},
		) {
			continue
		}

		typeObj := pass.TypesInfo.ObjectOf(typeSpec.Name)
		if typeObj == nil {
			continue
		}

		pass.ExportObjectFact(typeObj, &ConstructorFact{
			ConstructorName: constructorName,
			Pos:             constructorPos,
			End:             constructorEnd,
		})

		return
	}

	pass.Report(analysis.Diagnostic{
		Pos:     genDecl.Pos(),
		End:     genDecl.End(),
		Message: fmt.Sprintf("Constructor %q does not return the corresponding type", constructorName),
	})
}

func (l *linter) processCallExpr(pass *analysis.Pass, callExpr *ast.CallExpr, data *data) {
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

	data.zeroValues = append(data.zeroValues, initedEntry{
		pos: callExpr.Pos(),
		typ: namedTypeObj,
	})
}

func (l *linter) processCompositeLit(pass *analysis.Pass, compositeLit *ast.CompositeLit, data *data) {
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

	if compositeLit.Elts == nil {
		data.zeroValues = append(data.zeroValues, initedEntry{
			pos: compositeLit.Pos(),
			typ: namedTypeObj,
		})

		return
	}

	data.compositeLiterals = append(data.compositeLiterals, initedEntry{
		pos: compositeLit.Pos(),
		typ: namedTypeObj,
	})
}

func (l *linter) processValueSpec(pass *analysis.Pass, valueSpec *ast.ValueSpec, data *data) {
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

		data.zeroValues = append(data.zeroValues, initedEntry{
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

		data.nilValues = append(data.nilValues, initedEntry{
			pos: valueSpec.Pos(),
			typ: namedTypeObj,
		})
	}
}

var directiveRegex = regexp.MustCompile(`#constructor\[([^\]\r\n]+)\]`)

func constructorNameFromDocLine(docLine string) string {
	docLine = strings.TrimPrefix(docLine, "// ")
	docLine = strings.TrimSpace(docLine)
	if docLine == "" {
		return ""
	}

	m := directiveRegex.FindStringSubmatch(docLine)
	if len(m) != 2 {
		return ""
	}

	constructorName := m[1]

	return constructorName
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
