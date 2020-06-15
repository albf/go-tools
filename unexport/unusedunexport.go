// Package unusedunexport looks for unused unexported:
// - global constants and variables (type assertions are ignored).
// - structs and fields (assign-only is considered non-used).
// - interfaces and its methods.
// - functions, its arguments and return values (except main/init and cgo functions).
// - methods and, as functions, arguments/return values (methods that match interfaces are ignored).
package unusedunexport

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/container/intsets"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"
	"bitbucket.org/creachadair/stringset"
)

// Analyzer implements the Analyzer interface.
var Analyzer = &analysis.Analyzer{
	Name:     "unusedunexport",
	Doc:      "check unused unexported: global variables, constants, structs, fields methods, functions and their arguments and return values",
	Run:      run,
	Requires: []*analysis.Analyzer{buildssa.Analyzer},
}

type functionEntry struct {
	// Function full name as described by TypesInfo.
	fullName string

	// If there's at least one reference within the whole package.
	referenced bool

	// Declaration position, to identify usages.
	pos token.Pos

	// Unused results, stored by their return positions.
	unusedResults intsets.Sparse

	// Unused arguments, stored by their names
	unusedArguments stringset.Set

	// Whether function is passed as argument to another function.
	passedAsArgument bool
}

var (
	ignoredFunctions = map[string]bool{
		"main": true,
		"init": true,
	}
)

type structEntry struct {
	// Function full name as described by TypesInfo.
	fullName string

	// If there's at least one reference within the whole package.
	referenced bool

	// Exported indicates whether or not the struct is exported.
	exported bool

	// Declaration position, to identify usages.
	pos token.Pos

	// Unused unexported fields with their position, used for reporting errors.
	unusedFields map[int]token.Pos
}

type executionMetadata struct {
	// Function Registry that keeps an registry of all unexported functions and methods.
	functionRegistry map[string]*functionEntry

	// Struct Registry that keeps an registry of all unexported structs.
	structRegistry map[string]*structEntry

	// Holds receiver positions, to avoid false negatives on struct usages.
	receiverPositions map[token.Pos]bool

	// Other definitions hold unexported constants, variables and interfaces.
	otherUnusedDefinitions map[token.Pos]string
	pass                   *analysis.Pass
}

func newExecutionMetadata(pass *analysis.Pass) *executionMetadata {
	return &executionMetadata{
		functionRegistry:       make(map[string]*functionEntry),
		structRegistry:         make(map[string]*structEntry),
		receiverPositions:      make(map[token.Pos]bool),
		otherUnusedDefinitions: make(map[token.Pos]string),
		pass:                   pass,
	}
}

// run reports statements that copy values of certain problematic types.
func run(pass *analysis.Pass) (interface{}, error) {
	em := newExecutionMetadata(pass)
	registerPackageUnexportedDefinitions(em)

	ssainput := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)
	if initFunc := ssainput.Pkg.Func("init"); initFunc != nil {
		checkRegisteredFunctionAndFieldsUsages(em, initFunc)
	}

	for _, fn := range ssainput.SrcFuncs {
		fnObj, ok := fn.Object().(*types.Func)
		if !ok {
			continue
		}

		// Check for all existing calls matching registered functions.
		checkRegisteredFunctionAndFieldsUsages(em, fn)

		// Check for unused parameters, only for functions initially registered.
		fe := em.functionRegistry[fnObj.FullName()]
		if fe == nil || fe.passedAsArgument {
			continue
		}

		for _, param := range fn.Params {
			if (param.Referrers() == nil) || len(*param.Referrers()) == 0 {
				fe.unusedArguments.Add(param.Name())
			}
		}
	}

	// Check for other (non-function/methods) unused unexported definitions.
	visit := func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Ident:
			// Ignore the definition itself.
			if n.Obj != nil && n.Pos() == n.Obj.Pos() {
				return true
			}

			identObj := em.pass.TypesInfo.ObjectOf(n)
			if identObj == nil {
				return true
			}

			// Check for struct usages.
			if typeName, ok := identObj.(*types.TypeName); ok {
				if se, ok := em.structRegistry[typeName.Type().String()]; ok {
					// Discard receiver usages.
					if em.receiverPositions[n.Pos()] {
						return true
					}

					se.referenced = true
					return true
				}
			}

			// Fallback into other unexported definitions. Functions/methods are checked by ssa.
			delete(em.otherUnusedDefinitions, identObj.Pos())
		}
		return true
	}

	for _, f := range pass.Files {
		ast.Inspect(f, visit)
	}

	reportErrors(em)
	return nil, nil
}

func checkRegisteredFunctionAndFieldsUsages(em *executionMetadata, fn *ssa.Function) {
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			switch v := instr.(type) {
			// Look for function/method usages, including returned values.
			case *ssa.Call:
				checkFunctionParameter(em, v.Common())
				fe, ok := retrieveRegisteredFunctionEntry(em, v.Common())
				if !ok {
					break
				}
				for _, resIndex := range findUsedResults(v) {
					fe.unusedResults.Remove(resIndex)
				}
			case *ssa.Defer:
				checkFunctionParameter(em, v.Common())
				retrieveRegisteredFunctionEntry(em, v.Common())
			case *ssa.Go:
				checkFunctionParameter(em, v.Common())
				retrieveRegisteredFunctionEntry(em, v.Common())
			// Check if function is changing its type, which indicates complicated usage.
			case *ssa.ChangeType:
				f, ok := v.X.(*ssa.Function)
				if ok {
					funcObj, ok := f.Object().(*types.Func)
					if ok {
						markFunctionPassedAsArgument(em, funcObj)
					}
				}

			// Check for field usages.
			case *ssa.FieldAddr:
				xType := v.X.Type()
				xTypeName := xType.String()
				if xPointer, ok := xType.(*types.Pointer); ok {
					xTypeName = xPointer.Elem().String()
				}

				se, ok := em.structRegistry[xTypeName]
				if !ok {
					break
				}

				// Check for referrers to find out if it's not an assign.
				if v.Referrers() != nil {
					for _, ref := range *v.Referrers() {
						// Field assign, not a valid use.
						if store, ok := ref.(*ssa.Store); ok && store.Addr == v {
							continue
						}
						delete(se.unusedFields, v.Field)
						break
					}
				}
			}
		}
	}

	// Check anon registered functions
	for _, anon := range fn.AnonFuncs {
		checkRegisteredFunctionAndFieldsUsages(em, anon)
	}
}

func checkFunctionParameter(em *executionMetadata, cc *ssa.CallCommon) {
	for _, arg := range cc.Args {
		var funcObj *types.Func
		switch n := arg.(type) {
		// Static function passed as an argument.
		case *ssa.Function:
			var ok bool
			funcObj, ok = n.Object().(*types.Func)
			if !ok {
				continue
			}
		// Method with receiver passed as an argument.
		case *ssa.MakeClosure:
			var ok bool
			cfunc, ok := n.Fn.(*ssa.Function)
			if !ok {
				continue
			}
			funcObj, ok = cfunc.Object().(*types.Func)
			if !ok {
				continue
			}
		}

		if funcObj != nil {
			markFunctionPassedAsArgument(em, funcObj)
		}
	}
}

func markFunctionPassedAsArgument(em *executionMetadata, funcObj *types.Func) {
	fe := em.functionRegistry[funcObj.FullName()]
	if fe == nil {
		return
	}

	// Function/method was passed as parameter, which makes it very hard. Assume everything was used.
	fe.referenced = true
	fe.unusedResults.Clear()
	for k := range fe.unusedArguments {
		delete(fe.unusedArguments, k)
	}
	fe.passedAsArgument = true
}

func retrieveRegisteredFunctionEntry(em *executionMetadata, cc *ssa.CallCommon) (*functionEntry, bool) {
	var funName string
	staticCallee := cc.StaticCallee()

	switch {
	case staticCallee != nil && staticCallee.Object() != nil:
		funName = staticCallee.Object().(*types.Func).FullName()
	case cc.Method != nil:
		funName = cc.Method.FullName()
	default:
		return nil, false
	}

	fe := em.functionRegistry[funName]
	if fe == nil {
		return nil, false
	}
	fe.referenced = true
	return fe, true
}

func findUsedResults(v *ssa.Call) []int {
	var used []int
	refs := v.Referrers()
	numReturns := v.Call.Signature().Results().Len()

	switch numReturns {
	case 0:
		// Nothing to be used.
	case 1:
		if refs != nil && len(*refs) > 0 {
			used = append(used, 0)
		}
	default:
		if refs == nil {
			break
		}

		for _, ref := range *refs {
			extract, ok := ref.(*ssa.Extract)
			if !ok {
				continue
			}

			extractRefs := extract.Referrers()
			if extractRefs != nil && len(*extractRefs) > 0 {
				used = append(used, extract.Index)
			}
		}
	}

	return used
}

func registerPackageUnexportedDefinitions(em *executionMetadata) {
	interfaceMethods := make(map[string]bool)

	// Find all unexported global variables, constants, structs and interface definitions.
	// Interfaces method names are ignored, to avoid internal interface conflicts.
	for _, file := range em.pass.Files {
		for _, d := range file.Decls {
			gen, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, s := range gen.Specs {
				switch spec := s.(type) {
				case *ast.TypeSpec:
					switch inter := spec.Type.(type) {
					case *ast.StructType:
						registerStruct(em, spec, inter)
					case *ast.InterfaceType:
						if !spec.Name.IsExported() {
							em.otherUnusedDefinitions[spec.Name.Pos()] = spec.Name.Name
						}

						for _, m := range inter.Methods.List {
							funcType, ok := m.Type.(*ast.FuncType)
							if !ok {
								continue
							}

							for _, n := range m.Names {
								if n.IsExported() {
									continue
								}

								// Unexported interface method, ignore functions with the same name but register itself.
								interfaceMethods[n.Name] = true
								registerInterfaceMethod(em, funcType, n)
							}
						}
					}
				case *ast.ValueSpec:
					for _, name := range spec.Names {
						if !name.IsExported() && name.Name != "_" {
							em.otherUnusedDefinitions[name.Pos()] = name.Name
						}
					}
				}
			}
		}
	}

	for _, file := range em.pass.Files {
		for _, d := range file.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Name.IsExported() {
				continue
			}
			if shouldSkipFunction(fn) {
				continue
			}
			funcObj, ok := em.pass.TypesInfo.ObjectOf(fn.Name).(*types.Func)
			if !ok {
				continue
			}
			funcName := funcObj.Name()

			// Only ignore methods (with receiver) based on interface methods.
			if fn.Recv != nil && interfaceMethods[funcName] {
				continue
			}

			registerFunction(em, funcObj, fn.Type.Results.NumFields(), fn.Type.Params.List)

			// Register receivers for safely discarding them from usages.
			if fn.Recv != nil && len(fn.Recv.List) > 0 {
				em.receiverPositions[fn.Recv.List[0].Type.Pos()] = true
			}
		}
	}
}

func shouldSkipFunction(fn *ast.FuncDecl) bool {
	funcName := fn.Name.Name
	if ignoredFunctions[funcName] {
		return true
	}

	// Ignore cgo functions.
	lowerCase := strings.ToLower(funcName)
	if strings.HasPrefix(lowerCase, "_cgo") ||
		strings.HasPrefix(lowerCase, "_cfunc_") {
		return true
	}
	return false
}

func registerStruct(em *executionMetadata, spec *ast.TypeSpec, inter *ast.StructType) {
	typeNameObj, ok := em.pass.TypesInfo.ObjectOf(spec.Name).(*types.TypeName)
	if !ok {
		return
	}

	fullName := typeNameObj.Type().String()
	se := &structEntry{
		fullName:     fullName,
		exported:     spec.Name.IsExported(),
		pos:          spec.Name.Pos(),
		unusedFields: make(map[int]token.Pos),
	}

	if inter.Fields != nil {
		fieldIndex := 0
		for _, field := range inter.Fields.List {
			if field.Names != nil {
				for _, name := range field.Names {
					idx := fieldIndex
					fieldIndex++

					if name.IsExported() {
						continue
					}

					se.unusedFields[idx] = name.Pos()
				}
			} else {
				// Base struct, still a valid field but we won't check it here.
				fieldIndex++
			}
		}
	}

	em.structRegistry[fullName] = se
}

func registerInterfaceMethod(em *executionMetadata, funcType *ast.FuncType, n *ast.Ident) {
	funcObj, ok := em.pass.TypesInfo.ObjectOf(n).(*types.Func)
	if !ok {
		return
	}
	var numResults int
	if funcType.Results != nil {
		numResults = len(funcType.Results.List)
	}
	// Arguments are not checked for interface methods.
	registerFunction(em, funcObj, numResults, nil)
}

func registerFunction(em *executionMetadata, funcObj *types.Func, numResults int, params []*ast.Field) {
	// Found a well formed package-level function, initialize all returns
	// are unused and register as a function of interest.
	functionEntry := &functionEntry{
		fullName:        funcObj.FullName(),
		pos:             funcObj.Pos(),
		unusedResults:   intsets.Sparse{},
		unusedArguments: stringset.Set{},
	}
	for i := 0; i < numResults; i++ {
		functionEntry.unusedResults.Insert(i)
	}
	em.functionRegistry[funcObj.FullName()] = functionEntry
}

func reportErrors(em *executionMetadata) {
	for _, se := range em.structRegistry {
		if !se.referenced && !se.exported {
			em.pass.Reportf(se.pos, "Struct %v is not used or exported", se.fullName)
		}

		for _, pos := range se.unusedFields {
			em.pass.Reportf(pos, "Field is not used or exported")
		}
	}

	for _, fe := range em.functionRegistry {
		switch {
		case !fe.referenced:
			em.pass.Reportf(fe.pos, "Function %v is not used or exported", fe.fullName)
		case !fe.unusedResults.IsEmpty():
			em.pass.Reportf(fe.pos, "Function %v has unused returns with indexes: %v", fe.fullName, fe.unusedResults.String())
		case !fe.unusedArguments.Empty():
			em.pass.Reportf(fe.pos, "Function %v has unused arguments: %v", fe.fullName, fe.unusedArguments)
		}
	}

	for pos, name := range em.otherUnusedDefinitions {
		em.pass.Reportf(pos, "%v is not used or exported", name)
	}
}

