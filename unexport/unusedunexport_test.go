package unusedunexport

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyze(t *testing.T) {
	unusedMethodsSrc := `
package main

import (
	"fmt"
)

func usedFunc() int {
	return 5
}

func usedFuncInExtraFile() int {
	return 5
}

func usedFuncAssign() int {
	return 5
}

func unusedFunc() int { // want "not used or exported"
	return 5 + usedFunc()
}

func usedFuncNoResults() {
	return
}

type myReturn struct {
	value bool
}

func usedFuncStructResult() myReturn {
	return myReturn{}
}

func unusedFuncNoResults() { // want "not used or exported"
	fmt.Println("Never Called")
}

func ignoredResult() int { // want "has unused returns with indexes"
	return 10
}

func ignoredResultReassign() int { // want "has unused returns with indexes"
	return 10
}

func ignoredResultAssign() int { // want "has unused returns with indexes"
	return 10
}

func ignoredResultMultipleAssign() (int, int) { // want "has unused returns with indexes"
	return 10, 10
}

func multiAssignUsed() int {
	return 10
}

func multiAssignUsedMix() int {
	return 10
}

func multiAssignUnused() int { // want "has unused returns with indexes"
	return 10
}

func multiAssignMultipleCalls() (int, int) {
	return 10, 10
}

func multiAssignConcurrentAssign() (int, int) {
	return 10, 10
}

func longListOfReturnsOneIgnored() (int, int, int, int, int) { // want "has unused returns with indexes"
	return 10, 10, 10, 10, 10
}

type valueHolder struct {
	value int
}

func (vh valueHolder) myValue() int {
	return vh.value
}

func (vh valueHolder) myValueIgnoredResult() int { // want "has unused returns with indexes"
	return vh.value
}

func (vh *valueHolder) myValuePointer() int { // want "has unused returns with indexes"
	return vh.value
}

func (vh *valueHolder) myValueUnused() int { // want "not used or exported"
	return vh.value
}

func (vh *valueHolder) myValueIgnoredResultMultipleAssign() (int, int) { // want "has unused returns with indexes"
	return vh.value, vh.value
}

func (vh *valueHolder) myValueMultiAssignUsed() int {
	return vh.value
}

func (vh *valueHolder) myValueMultiAssignUnused() int { // want "has unused returns with indexes"
	return vh.value
}

func (vh *valueHolder) myValueMultiAssignMultipleCalls() (int, int) {
	return vh.value, vh.value
}

type valueHolderPointer struct {
	value int
	unused int // want "not used or exported"
}

func (vh *valueHolderPointer) myValue() int {
	return vh.value
}

type valueHolderAnonFunc struct {
	value int
}

func (vh *valueHolderAnonFunc) myValue() int {
	return vh.value
}

func anonCaller(f func()) {
	f()
}

type myInterface interface {
	interfaceMethod() string
	interfaceMethodUnused() string // want "not used or exported"
	interfaceMethodWithArg(arg string) string
}

func (vh valueHolder) interfaceMethod() string {
	return "should be ignored"
}

func (vh *valueHolder) interfaceMethodUnused() string {
	return "should be ignored"
}

func interfaceMethod() string { // want "not used or exported"
	return "no receiver, should be marked as unused"
}

type interfaceImplementation struct{}

func (ii interfaceImplementation) interfaceMethod() string {
	return "should be ignored"
}

func (ii interfaceImplementation) interfaceMethodUnused() string {
	return "should be ignored"
}

func (ii interfaceImplementation) interfaceMethodWithArg(arg string) string {
	return arg
}

func deferMethodNoResults() {
	fmt.Println("should be fine")
}

func deferMethodWithReturn() string { // want "has unused returns with indexes"
	return "return value is ignored"
}

func _cgofunction() string {
	return "_cgo functions should be ignored"
}

func _Cgo_function() string {
	return "_cgo functions should be ignored"
}

func goCallFunction() string { // want "has unused returns with indexes"
	return "value is also ignored"
}

func init() {
	fmt.Println("init should be ignored")
	fmt.Println(wrapper(wrappedFunctionInit, 10))
}

func unusedArg(myArg int) string { // want "unused arguments"
	return "arg ignored"
}

func usedArg(arg string) string {
	return arg + "abc"
}

func unusedArgWithCall(arg string) string { // want "unused arguments"
	return usedArg("some other value")
}

func usedArgWithCall(arg string) string {
	return usedArg(arg + "extra")
}

func unusedArgReassignedByIf(arg int) string { // want "unused arguments"
	if arg := usedFunc(); arg == 5 {
		return "arg is not really used"
	}
	return "unreachable"
}

func usedArgReassignedByIf(arg string) string {
	if arg := usedArg(arg); arg == "abc" {
		return "arg is actually used"
	}
	return "always returned"
}

func unusedArgReassigned(arg string) string { // want "unused arguments"
	arg = "abc"
	return arg
}

func unusedArgNested(arg string) string { // want "unused arguments"
	if arg := usedFunc(); arg == 5 {
		return "redefine first"
	}

	if usedFunc() > 10 {
		arg = "re-assign, still not used"
		return arg
	}
	return "default"
}

func usedArgNested(arg string) string {
	if arg := usedFunc(); arg == 5 {
		return "redefine first"
	}

	if usedFunc() > 10 {
		arg = "re-assign, still not used"
		return arg
	}

	// Finally used
	if usedFunc() < 10 {
		return arg
	}
	return "default"
}

func usedArgReassigned(arg string) string {
	arg = usedArg(arg)
	return arg
}

func usedArgSwitchInterface(mi myInterface) string {
	switch arg := mi.(type) {
	case interfaceImplementation:
		return arg.interfaceMethod()
	default:
		return "unkonwn"
	}
}

func usedArgSwitchInterfaceSameName(mi myInterface) string {
	switch mi := mi.(type) {
	case interfaceImplementation:
		return mi.interfaceMethod()
	default:
		return "unkonwn"
	}
}

func unusedArgSwitchInterfaceSimple(mi myInterface, arg string) string { // want "unused arguments"
	switch v := mi.(type) {
	case interfaceImplementation:
		return v.interfaceMethod()
	default:
		return "unkonwn"
	}
}

func unusedArgSwitchInterface(mi myInterface, arg string) string { // want "unused arguments"
	switch arg := mi.(type) {
	case interfaceImplementation:
		return arg.interfaceMethod()
	default:
		return "unkonwn"
	}
}

func usedArgSwitchInterfaceReassign(mi myInterface, arg string) string {
	switch arg := mi.(type) {
	case interfaceImplementation:
		return arg.interfaceMethod()
	}
	return arg
}

func usedArgElse(arg string) string {
	var res string
	if usedFunc() > 10 {
		res = "higher"
	} else {
		res = arg
	}
	return res
}

func unusedArgElseReassign(arg int) int { // want "unused arguments"
	var res int
	if arg := usedFunc(); arg > 10 {
		res = 10
	} else {
		res = arg
	}
	return res
}

func usedArgElseWithIfReassign(arg int) int {
	var res int
	if usedFunc() > 10 {
		arg = 10
		res = arg
	} else {
		res = arg
	}
	return res
}

func usedArgElseIf(arg int) int {
	if x := usedFunc(); x > 10 {
		return 10
	} else if x > 5 {
		return arg
	} else {
		return x
	}
}

func unusedArgElseIf(arg int) int { // want "unused arguments"
	if arg := usedFunc(); arg > 10 {
		return 10
	} else if arg > 5 {
		return arg
	} else {
		return 1
	}
}

func usedArgForLoop(arg int) int {
	var res int
	for i := 0; i < arg; i++ {
		res += i
	}
	return res
}

// Assert global declarations don't get in the way.
var loopUnused int

func unusedArgForLoopReassign(loopUnused int) int { // want "unused arguments"
	var res int
	for loopUnused := 0; loopUnused < 10; loopUnused++ {
		res += loopUnused
	}
	return res
}

func unusedArgInnerBlockDeclaration(arg string) string { // want "unused arguments"
	if usedFunc() > 10 {
		var arg int
		return fmt.Sprint(arg)
	}
	return "argument is not used"
}

func unusedArgInnerBlockDeclarationMultiple(arg, other string) string { // want "unused arguments"
	if arg == "abc" {
		return arg
	}
	{
		var arg, other int
		return fmt.Sprint(arg, other)
	}
}

func copiedFunction(value int) int {
	return 10 * value
}

func wrappedFunction(value int) int {
	return 10 * value
}

func wrappedFunctionDefer(value int) int {
	return 10 * value
}

func wrappedFunctionGo(value int) int {
	return 10 * value
}

func wrappedFunctionInit(_ int) int {
	return 10
}

func WrappedFunctionExported(_ int) int {
	return 10
}

type wrappedStruct struct {
	internalValue int
}

func (u *wrappedStruct) wrappedFunction(value int) int {
	return 10 * value * u.internalValue
}

func wrapper(f func(value int) int, v int) int {
	return f(v)
}

type myFunctionType func(value int) int

func myFunctionTypeCase(value int) int {
	return 10
}

type myFunctionTypeGlobal func(value int) int

func myFunctionTypeCaseGlobal(value int) int {
	return 10
}

var functionGroupGlobal = map[string]myFunctionTypeGlobal {
	"myFunc":  myFunctionTypeCaseGlobal,
}

func usedOnlyByGlobalVar(value int) int {
	return value * 10
}

var unusedGlobalFunction = usedOnlyByGlobalVar(20) // want "not used or exported"

func main() {
	// Try to trick parser by renaming local "unusedFunc" identifier.
	var unusedFunc int
	unusedFunc = 2
	fmt.Println(unusedFunc)

	// Call method but ignore return values.
	ignoredResult()
	_ = ignoredResultAssign()
	assignMe_0, _ := ignoredResultMultipleAssign()
	assignMe_1, _ := multiAssignUsed(), multiAssignUnused()
	_, decoy, assignMe_2 := multiAssignUsedMix(), fmt.Sprintf("abc"), multiAssignUsedMix()

	// Different complementary calls
	assignMe_Mul_0, _ := multiAssignMultipleCalls()
	_, assignMe_Mul_1 := multiAssignMultipleCalls()

	// Methods with receivers
	vh := valueHolder{}
	fmt.Println(vh.myValue())
	vhp := &valueHolderPointer{}
	fmt.Println(vhp.myValue())

	// Anon Funcs
	anonCaller(func() {
		var x valueHolderAnonFunc
		fmt.Println(x.myValue())
	})

	// Call method but ignore return values.
	vh.myValuePointer()
	vh.myValueIgnoredResult()
	assignMe_3, _ := vh.myValueIgnoredResultMultipleAssign()
	assignMe_4, _ := vh.myValueMultiAssignUsed(), vh.myValueMultiAssignUnused()

	// Different complementary calls
	assignMe_Mul_2, _ := vh.myValueMultiAssignMultipleCalls()
	_, assignMe_Mul_3 := vh.myValueMultiAssignMultipleCalls()

	// Concurrent assign call
	assignMe_Mul_4, assignMe_Mul_5 := multiAssignConcurrentAssign()
	assignMe_Mul_6, assignMe_Mul_7, assignMe_Mul_8, _, assignMe_Mul_9 := longListOfReturnsOneIgnored()

	// Interface implementation
	var i myInterface
	i = interfaceImplementation{}
	fmt.Println(i.interfaceMethod())
	fmt.Println(i.interfaceMethodWithArg("dummy"))

	// Defers
	defer deferMethodNoResults()
	defer deferMethodWithReturn()

	// Go call
	go goCallFunction()

	// Assign/No return or ignored initial assign.
	var assignMe_5 int
	assignMe_5 = ignoredResultReassign()
	assignMe_5 = usedFuncAssign()
	structRes := usedFuncStructResult()
	usedFuncNoResults()

	// Print used funcs/values.
	fmt.Println(usedFunc())
	fmt.Println(assignMe_0)
	fmt.Println(assignMe_1)
	fmt.Println(assignMe_2)
	fmt.Println(assignMe_3)
	fmt.Println(assignMe_4)
	fmt.Println(assignMe_5)
	fmt.Println(decoy)
	fmt.Println(assignMe_Mul_0)
	fmt.Println(assignMe_Mul_1)
	fmt.Println(assignMe_Mul_2)
	fmt.Println(assignMe_Mul_3)
	fmt.Println(assignMe_Mul_4)
	fmt.Println(assignMe_Mul_5)
	fmt.Println(assignMe_Mul_6)
	fmt.Println(assignMe_Mul_7)
	fmt.Println(assignMe_Mul_8)
	fmt.Println(assignMe_Mul_9)
	fmt.Println(structRes)
	fmt.Println(structRes.value)

	// Unused Arguments
	fmt.Println(unusedArg(10))
	fmt.Println(unusedArgWithCall("dummy"))
	fmt.Println(usedArg("dummy"))
	fmt.Println(usedArgWithCall("dummy"))
	fmt.Println(unusedArgReassignedByIf(10))
	fmt.Println(usedArgReassignedByIf("dummy"))
	fmt.Println(unusedArgReassigned("dummy"))
	fmt.Println(unusedArgNested("dummy"))
	fmt.Println(usedArgNested("dummy"))
	fmt.Println(usedArgReassigned("dummy"))
	fmt.Println(usedArgSwitchInterface(i))
	fmt.Println(usedArgSwitchInterfaceSameName(i))
	fmt.Println(unusedArgSwitchInterfaceSimple(i, "dummy"))
	fmt.Println(unusedArgSwitchInterface(i, "dummy"))
	fmt.Println(usedArgSwitchInterfaceReassign(i, "dummy"))
	fmt.Println(usedArgElse("dummy"))
	fmt.Println(unusedArgElseReassign(10))
	fmt.Println(usedArgElseWithIfReassign(10))
	fmt.Println(usedArgElseIf(10))
	fmt.Println(unusedArgElseIf(10))
	fmt.Println(usedArgForLoop(10))
	fmt.Println(loopUnused)
	fmt.Println(unusedArgForLoopReassign(10))
	fmt.Println(unusedArgInnerBlockDeclaration("dummy"))
	fmt.Println(unusedArgInnerBlockDeclarationMultiple("dummy", "dummy"))

	// Call functions by copying its address
	copyFunction := copiedFunction
	fmt.Println(copyFunction(10))
	fmt.Println(wrapper(wrappedFunction, 10))
	defer wrapper(wrappedFunctionDefer, 10)
	go wrapper(wrappedFunctionGo, 10)
	fmt.Println(wrapper(WrappedFunctionExported, 10))

	wsi := &wrappedStruct{internalValue: 10}
	fmt.Println(wrapper(wsi.wrappedFunction, 10))

	// Typed function usage, should assume everything is used.
	var functionGroup = map[string]myFunctionType{
		"myFunc":  myFunctionTypeCase,
	}
	for _, afg := range functionGroup {
		fmt.Println(afg(10))
	}
	for _, afg := range functionGroupGlobal {
		fmt.Println(afg(10))
	}

	// Call unused definitions helper function to avoid unused functions.
	otherUnusedDefinitionsHelper()
	structHelper()
}
`

	unusedStructSrc := `
package main

import (
	"fmt"
)

type usedType struct {
	value bool
}

type usedTypeInExtraFile struct {
	intValue int
}

type usedFieldWithinMethod struct {
	value bool
}

func (u usedFieldWithinMethod) myValue(x bool) bool {
	return x && u.value
}

type unusedField struct {
	ExportedValue0 bool
	value bool           // want "not used or exported"
	ExportedValue1 bool
}

type assignFieldOnly struct {
	value bool           // want "not used or exported"
}

type initializationOnly struct {
	value bool           // want "not used or exported"
}

type unusedType struct { // want "not used or exported"
	value bool           // want "not used or exported"
}

type UnusedExportedType struct {
	value bool           // want "not used or exported"
}

type usedExportedType struct {
	value bool           // want "not used or exported"
}

type compositeUsed struct {
	value *compositeInner
	valueWithMethod *compositeInnerWithMethod
}

type compositeInner struct {
	innerValue bool
}

func (cu compositeUsed) isEqualInner(x bool) bool {
	return cu.value.innerValue == x
}

type compositeInnerWithMethod struct {
	innerValue bool
}

func (ciwm *compositeInnerWithMethod) isEqualInner(x bool) bool {
	return ciwm.innerValue == x
}

type usedBaseStruct struct{}

func (ubs *usedBaseStruct) ExportedMethod(x bool) bool {
	return !x
}

type usedTypeWithBase struct {
	usedBaseStruct
	valueBeingUsed bool
}

type unusedFieldWithUsedExport struct {
	ExportedValue bool
	ignoredValue bool // want "not used or exported"
}

type unusedTypeWithMethod struct { // want "not used or exported"
	myValue bool
}

func (u unusedTypeWithMethod) value0(x bool) int {
	return u.value1(!x && u.myValue)
}

func (u unusedTypeWithMethod) value1(x bool) int {
	return u.value0(!x && u.myValue)
}

type channelStruct struct{
	usedValue bool
}

type channelStructUnusedField struct{
	unusedValue bool  // want "not used or exported"
}

func structHelper() {
	var ut usedType
	fmt.Println(ut)
	fmt.Println(ut.value)

	var ufwm usedFieldWithinMethod
	fmt.Println(ufwm.myValue(true))

	var uf unusedField
	fmt.Println(uf)

	var afo assignFieldOnly
	afo.value = false
	fmt.Println(afo)

	inito := initializationOnly{value: false}
	fmt.Println(inito)

	var uet usedExportedType
	fmt.Println(uet)

	cu := compositeUsed{
		value: &compositeInner { innerValue: true },
		valueWithMethod: &compositeInnerWithMethod { innerValue: true },
	}
	fmt.Println(cu.isEqualInner(true))
	fmt.Println(cu.valueWithMethod.isEqualInner(true))

	utwb := usedTypeWithBase{}
	fmt.Println(utwb.valueBeingUsed)

	ufwue := unusedFieldWithUsedExport{}
	fmt.Println(ufwue.ExportedValue)

	usedChannel := make(chan channelStruct)
	cs := <-usedChannel
	fmt.Println(cs.usedValue)

	unusedFieldChannel := make(chan channelStructUnusedField)
	unusedFieldChannel <- channelStructUnusedField{unusedValue: true}
}
`

	otherUnusedDefinitionsSrc := `
package main

import (
	"fmt"
)

const usedConst = 10
const usedConstInExtraFile = 10
const unusedConst = 20 // want "not used or exported"

var usedGlobalVar int
var unusedGlobalVar int // want "not used or exported"

var (
	multipleGlobalVar0, multipleGlobalVar1,multipleGlobalVar2 int
)

var (
	multipleGlobalVar3, multipleGlobalVar4,multipleGlobalVar5 int // want "not used or exported"
)

type usedInterface interface {
	value() int
}

type usedInterfaceImplementer struct {
	myvalue int
}

func (ut usedInterfaceImplementer) value() int {
	return ut.myvalue
}

type unusedInterface interface { // want "not used or exported"
	value() int                  // want "not used or exported"
}

type UnusedInterfaceExported interface {
	ValueExported() int
	value() int                  // want "not used or exported"
}

type assertedInterface interface {
	dummy() int                  // want "not used or exported"
}

type assertedInterfaceImplementer struct {
	myvalue int
}

func (aii *assertedInterfaceImplementer) dummy() int {
	return aii.myvalue
}

// Compliance compile-time type assertions.
var (
	_ = assertedInterface(&assertedInterfaceImplementer{})
	_ assertedInterface = (*assertedInterfaceImplementer)(nil)
)


func otherUnusedDefinitionsHelper() {
	fmt.Println(usedConst)
	fmt.Println(extraFileUsageFunc())
	fmt.Println(usedGlobalVar)
	fmt.Println(multipleGlobalVar0)
	fmt.Println(multipleGlobalVar1)
	fmt.Println(multipleGlobalVar2)
	fmt.Println(multipleGlobalVar3)
	fmt.Println(multipleGlobalVar4)

	var interfaceInstance usedInterface
	interfaceInstance = usedInterfaceImplementer{}
	fmt.Println(interfaceInstance.value())
}
`

	extraFileSrc := `
package main

func extraFileUsageFunc() int {
	var utief usedTypeInExtraFile
	utief.intValue = usedFuncInExtraFile()
	return usedConstInExtraFile + utief.intValue + 10
}
`

	files := map[string]string{
		"p/methods.go":     unusedMethodsSrc,
		"p/structs.go":     unusedStructSrc,
		"p/definitions.go": otherUnusedDefinitionsSrc,
		"p/extra.go":       extraFileSrc,
	}

	dir, cleanup, err := analysistest.WriteFiles(files)
	if err != nil {
		t.Fatal(err)
	}
	analysistest.Run(t, dir, Analyzer, "p")
	cleanup()
}

