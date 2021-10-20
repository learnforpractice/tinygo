package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fatih/color"
)

/*
	bool
	int8
	uint8
	int16
	uint16
	int32
	uint32
	int64
	uint64
	int128
	uint128
	varint32
	varuint32
	float32
	float64
	float128
	time_point
	time_point_sec
	block_timestamp_type
	name
	bytes
	string
	checksum160
	checksum256
	checksum512
	public_key
	signature
	symbol
	symbol_code
	asset
	extended_asset
*/

func char_to_symbol(c byte) byte {
	if c >= 'a' && c <= 'z' {
		return (c - 'a') + 6
	}

	if c >= '1' && c <= '5' {
		return (c - '1') + 1
	}
	return 0
}

func StringToName(str string) uint64 {
	length := len(str)
	value := uint64(0)

	for i := 0; i <= 12; i++ {
		c := uint64(0)
		if i < length && i <= 12 {
			c = uint64(char_to_symbol(str[i]))
		}
		if i < 12 {
			c &= 0x1f
			c <<= 64 - 5*(i+1)
		} else {
			c &= 0x0f
		}

		value |= c
	}

	return value
}

func NameToString(value uint64) string {
	charmap := []byte(".12345abcdefghijklmnopqrstuvwxyz")
	// 13 dots
	str := []byte{'.', '.', '.', '.', '.', '.', '.', '.', '.', '.', '.', '.', '.'}

	tmp := value
	for i := 0; i <= 12; i++ {
		var c byte
		if i == 0 {
			c = charmap[tmp&0x0f]
		} else {
			c = charmap[tmp&0x1f]
		}
		str[12-i] = c
		if i == 0 {
			tmp >>= 4
		} else {
			tmp >>= 5
		}
	}

	i := len(str) - 1
	for ; i >= 0; i-- {
		if str[i] != '.' {
			break
		}
	}
	return string(str[:i+1])
}

func abiTypes() []string {
	return []string{
		"bool",
		"int8",
		"uint8",
		"int16",
		"uint16",
		"int32",
		"uint32",
		"int64",
		"uint64",
		"int128",
		"uint128",
		"varint32",
		"varuint32",
		"float32",
		"float64",
		"float128",
		"time_point",
		"time_point_sec",
		"block_timestamp_type",
		"name",
		"bytes",
		"string",
		"checksum160",
		"checksum256",
		"checksum512",
		"public_key",
		"signature",
		"symbol",
		"symbol_code",
		"asset",
		"extended_asset",
	}
}

func _convertToAbiType(goType string) (string, bool) {
	switch goType {
	case "byte":
		return "uint8", true
	case "bool":
		return "bool", true
	case "int8":
		return "int8", true
	case "uint8":
		return "uint8", true
	case "int16":
		return "int16", true
	case "uint16":
		return "uint16", true
	case "int32":
		return "int32", true
	case "uint32":
		return "uint32", true
	case "int64":
		return "int64", true
	case "uint64":
		return "uint64", true
	case "string":
		return "string", true
	case "float32":
		return "float32", true
	case "float64":
		return "float64", true
	case "chain.VarInt32":
		return "varint32", true
	case "chain.VarUint32":
		return "varuint32", true
	case "chain.Int128":
		return "int128", true
	case "chain.Uint128":
		return "uint128", true
	case "chain.Float128":
		return "float128", true
	case "chain.Name":
		return "name", true
	case "chain.TimePoint":
		return "time_point", true
	case "chain.TimePointSec":
		return "time_point_sec", true
	case "chain.BlockTimestampType":
		return "block_timestamp_type", true
	case "chain.Checksum160":
		return "checksum160", true
	case "chain.Checksum256":
		return "checksum256", true
	case "chain.Uint256":
		return "checksum256", true
	case "chain.Checksum512":
		return "checksum512", true
	case "chain.PublicKey":
		return "public_key", true
	case "chain.Signature":
		return "signature", true
	case "chain.Symbol":
		return "symbol", true
	case "chain.SymbolCode":
		return "symbol_code", true
	case "chain.Asset":
		return "asset", true
	case "chain.ExtendedAsset":
		return "extended_asset", true
	default:
		return "", false
	}
}

const (
	TYPE_ARRAY = iota + 1
	TYPE_SLICE
	TYPE_POINTER
)

type StructMember struct {
	Name        string
	Type        string
	LeadingType int
	Pos         token.Pos
}

type ActionInfo struct {
	ActionName string
	FuncName   string
	StructName string
	Members    []StructMember
	IsNotify   bool
	Ignore     bool
}

type SecondaryIndexInfo struct {
	Type   string
	Name   string
	Getter string
	Setter string
}

const (
	NormalType = iota
	BinaryExtensionType
	OptionalType
)

//handle binary_extension and optional abi types
type SpecialAbiType struct {
	typ    int
	name   string
	member StructMember
}

type StructInfo struct {
	PackageName      string
	TableName        string
	Singleton        bool
	StructName       string
	Comment          string
	PrimaryKey       string
	SecondaryIndexes []SecondaryIndexInfo
	Members          []StructMember
}

type CodeGenerator struct {
	dirName         string
	currentFile     string
	contractName    string
	fset            *token.FileSet
	codeFile        *os.File
	actions         []ActionInfo
	structs         []StructInfo
	specialAbiTypes []SpecialAbiType
	structMap       map[string]*StructInfo

	hasMainFunc        bool
	abiStructsMap      map[string]*StructInfo
	actionMap          map[string]bool
	contractStructName string
	hasNewContractFunc bool
	abiTypeMap         map[string]bool
	indexTypeMap       map[string]bool
}

type ABITable struct {
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	IndexType string   `json:"index_type"`
	KeyNames  []string `json:"key_names"`
	KeyTypes  []string `json:"key_types"`
}

type ABIAction struct {
	Name              string `json:"name"`
	Type              string `json:"type"`
	RicardianContract string `json:"ricardian_contract"`
}

type ABIStructField struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type ABIStruct struct {
	Name   string           `json:"name"`
	Base   string           `json:"base"`
	Fields []ABIStructField `json:"fields"`
}

type ABI struct {
	Version          string      `json:"version"`
	Structs          []ABIStruct `json:"structs"`
	Types            []string    `json:"types"`
	Actions          []ABIAction `json:"actions"`
	Tables           []ABITable  `json:"tables"`
	RicardianClauses []string    `json:"ricardian_clauses"`
	Variants         []string    `json:"variants"`
	AbiExtensions    []string    `json:"abi_extensions"`
	ErrorMessages    []string    `json:"error_messages"`
}

func NewCodeGenerator() *CodeGenerator {
	t := &CodeGenerator{}
	t.structMap = make(map[string]*StructInfo)
	t.abiStructsMap = make(map[string]*StructInfo)

	t.actionMap = make(map[string]bool)
	t.abiTypeMap = make(map[string]bool)
	t.indexTypeMap = make(map[string]bool)
	for _, abiType := range abiTypes() {
		t.abiTypeMap[abiType] = true
	}

	for _, indexType := range []string{"IDX64", "IDX128", "IDX256", "IDXFloat64", "IDXFloat128"} {
		t.indexTypeMap[indexType] = true
	}

	return t
}

func (t *CodeGenerator) convertToAbiType(pos token.Pos, goType string) (string, error) {
	abiType, ok := _convertToAbiType(goType)
	if ok {
		return abiType, nil
	}

	// check if type is an abi struct
	if _, ok := t.abiStructsMap[goType]; ok {
		return goType, nil
	}
	msg := fmt.Sprintf("type %s can not be converted to an ABI type", goType)
	if goType == "Asset" || goType == "Symbol" || goType == "Name" {
		msg += fmt.Sprintf("\nDo you mean chain.%s?", goType)
	}
	panic(t.newError(pos, msg))
	return "", t.newError(pos, msg)
}

func (t *CodeGenerator) convertType(goType StructMember) (string, error) {
	typ := goType.Type
	var specialAbiType *SpecialAbiType
	//special case for []byte type
	if typ == "byte" && goType.LeadingType == TYPE_SLICE {
		return "bytes", nil
	}

	pos := goType.Pos
	for i := range t.specialAbiTypes {
		if t.specialAbiTypes[i].name == typ {
			specialAbiType = &t.specialAbiTypes[i]
			typ = specialAbiType.member.Type
			pos = specialAbiType.member.Pos
			break
		}
	}

	abiType, err := t.convertToAbiType(pos, typ)
	if err != nil {
		return "", err
	}

	if goType.LeadingType == TYPE_SLICE {
		// if abiType == "byte" {
		// 	return "bytes", nil
		// }
		abiType += "[]"
	}

	if specialAbiType != nil {
		if specialAbiType.typ == BinaryExtensionType {
			return abiType + "$", nil
		} else if specialAbiType.typ == OptionalType {
			return abiType + "?", nil
		} else {
			return "", fmt.Errorf("unknown special abi type %d", specialAbiType.typ)
		}
	} else {
		return abiType, nil
	}
}

func (t *CodeGenerator) newError(p token.Pos, format string, args ...interface{}) error {
	errMsg := fmt.Sprintf(format, args...)
	return errors.New(t.getLineInfo(p) + ":\n" + errMsg)
}

func (t *CodeGenerator) parseField(structName string, field *ast.Field, memberList *[]StructMember, isStructField bool, ignore bool) error {
	if ignore {
		_, ok := field.Type.(*ast.StarExpr)
		if !ok {
			_, ok = field.Type.(*ast.ArrayType)
			if !ok {
				errMsg := fmt.Sprintf("ignored action parameter %v not a pointer type", field.Names)
				return errors.New(errMsg)
			}
		}
	}

	switch fieldType := field.Type.(type) {
	case *ast.Ident:
		if field.Names != nil {
			for _, v := range field.Names {
				member := StructMember{}
				member.Pos = field.Pos()
				member.Name = v.Name
				member.Type = fieldType.Name
				*memberList = append(*memberList, member)
			}
		} else {
			//TODO: parse anonymous struct
			member := StructMember{}
			member.Pos = field.Pos()
			member.Name = ""
			member.Type = fieldType.Name
			*memberList = append(*memberList, member)
		}
	case *ast.ArrayType:
		var leadingType int
		if fieldType.Len != nil {
			log.Printf("++++++fixed array %v ignored in %s\n", field.Names, structName)
			return nil
			leadingType = TYPE_ARRAY
		} else {
			leadingType = TYPE_SLICE
		}
		//*ast.BasicLit
		switch v := fieldType.Elt.(type) {
		case *ast.Ident:
			for _, name := range field.Names {
				member := StructMember{}
				member.Pos = field.Pos()
				member.Name = name.Name
				member.Type = v.Name
				member.LeadingType = leadingType
				*memberList = append(*memberList, member)
			}
		case *ast.ArrayType:
			for _, name := range field.Names {
				if ident, ok := v.Elt.(*ast.Ident); ok {
					member := StructMember{}
					member.Pos = field.Pos()
					member.Name = name.Name
					member.Type = "[]" + ident.Name
					member.LeadingType = leadingType
					*memberList = append(*memberList, member)
				} else {
					errMsg := fmt.Sprintf("Unsupported field %s", name)
					return t.newError(field.Pos(), errMsg)
				}
			}
		case *ast.SelectorExpr:
			ident := v.X.(*ast.Ident)
			for _, name := range field.Names {
				member := StructMember{}
				member.Pos = field.Pos()
				member.Name = name.Name
				member.Type = ident.Name + "." + v.Sel.Name
				member.LeadingType = leadingType
				*memberList = append(*memberList, member)
			}
		default:
			errMsg := fmt.Sprintf("unsupported type: %T %v", v, field.Names)
			return t.newError(field.Pos(), errMsg)
		}
	case *ast.SelectorExpr:
		ident := fieldType.X.(*ast.Ident)
		for _, name := range field.Names {
			member := StructMember{}
			member.Pos = field.Pos()
			member.Name = name.Name
			member.Type = ident.Name + "." + fieldType.Sel.Name
			// member.IsArray = false
			*memberList = append(*memberList, member)
		}
	case *ast.StarExpr:
		//Do not parse pointer type in struct
		if isStructField {
			log.Printf("Warning: Pointer type %v in %s ignored\n", field.Names, structName)
			return nil
		}

		switch v2 := fieldType.X.(type) {
		case *ast.Ident:
			for _, name := range field.Names {
				member := StructMember{}
				member.Pos = field.Pos()
				member.Name = name.Name
				member.Type = v2.Name
				member.LeadingType = TYPE_POINTER
				*memberList = append(*memberList, member)
			}
		case *ast.SelectorExpr:
			switch x := v2.X.(type) {
			case *ast.Ident:
				for _, name := range field.Names {
					member := StructMember{}
					member.Pos = field.Pos()
					member.Name = name.Name
					member.Type = x.Name + "." + v2.Sel.Name
					member.LeadingType = TYPE_POINTER
					*memberList = append(*memberList, member)
				}
			default:
				panic(fmt.Sprintf("Unknown pointer type: %T %v", x, x))
			}
		default:
			panic("Unhandled pointer type:" + fmt.Sprintf("%[1]v %[1]T", v2))
		}
	default:
		return t.newError(field.Pos(), "Unsupported field: %v", field.Names)
	}
	return nil
}

func (t *CodeGenerator) parseSpecialAbiType(packageName string, v *ast.GenDecl) bool {
	extension := SpecialAbiType{}
	if len(v.Specs) != 1 {
		return false
	}

	spec, ok := v.Specs[0].(*ast.TypeSpec)
	if !ok {
		return false
	}

	_struct, ok := spec.Type.(*ast.StructType)
	if !ok {
		return false
	}

	if _struct.Fields == nil || len(_struct.Fields.List) != 2 {
		return false
	}

	extension.name = spec.Name.Name
	field1 := _struct.Fields.List[0]
	if len(field1.Names) != 0 {
		return false
	}

	typ, ok := field1.Type.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident := typ.X.(*ast.Ident)
	if ident.Name+"."+typ.Sel.Name == "chain.BinaryExtension" {
		extension.typ = BinaryExtensionType
	} else if ident.Name+"."+typ.Sel.Name == "chain.Optional" {
		extension.typ = OptionalType
	} else {
		return false
	}

	field2 := _struct.Fields.List[1]
	if len(field2.Names) != 1 {
		return false
	}

	extension.member.Name = field2.Names[0].Name

	switch typ := field2.Type.(type) {
	case *ast.Ident:
		extension.member.Type = typ.Name
		extension.member.Pos = typ.Pos()
	case *ast.SelectorExpr:
		ident := typ.X.(*ast.Ident)
		extension.member.Type = ident.Name + "." + typ.Sel.Name
		extension.member.Pos = typ.Pos()
	default:
		// err := t.newError(v.Pos(), "Unsupported type: %[1]T %[1]v", typ)
		//panic(err)
		return false
	}
	t.specialAbiTypes = append(t.specialAbiTypes, extension)
	return true
}

func splitAndTrimSpace(s string, sep string) []string {
	parts := strings.Split(strings.TrimSpace(s), sep)
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func (t *CodeGenerator) parseStruct(packageName string, v *ast.GenDecl) error {
	if v.Tok != token.TYPE {
		return nil
	}

	if t.parseSpecialAbiType(packageName, v) {
		return nil
	}

	info := StructInfo{}
	info.PackageName = packageName
	isContractStruct := false
	var lastLineDoc string
	if v.Doc != nil {
		n := len(v.Doc.List)
		doc := v.Doc.List[n-1]
		lastLineDoc = strings.TrimSpace(doc.Text)
		if strings.HasPrefix(lastLineDoc, "//table") {
			//items := Split(lastLineDoc)
			parts := strings.Fields(lastLineDoc)
			if len(parts) == 2 && (parts[0] == "//table") {
				tableName := parts[1]
				if !IsNameValid(tableName) {
					return t.newError(doc.Pos(), "Invalid table name:"+tableName)
				}
				info.TableName = parts[1]
			} else if (len(parts) == 3) && (parts[0] == "//table") {
				tableName := parts[1]
				if !IsNameValid(tableName) {
					return t.newError(doc.Pos(), "Invalid table name:"+tableName)
				}
				info.TableName = parts[1]
				if parts[2] == "singleton" {
					info.Singleton = true
				}
			}
		} else if strings.HasPrefix(lastLineDoc, "//contract") {
			parts := strings.Fields(lastLineDoc)
			if len(parts) == 2 {
				name := parts[1]
				if t.contractName != "" {
					errMsg := fmt.Sprintf("contract name %s replace by %s", t.contractName, name)
					return t.newError(doc.Pos(), errMsg)
				}
				t.contractName = name
				isContractStruct = true
			}
		}
	}

	for _, spec := range v.Specs {
		v := spec.(*ast.TypeSpec)
		name := v.Name.Name
		if isContractStruct {
			t.contractStructName = name
		}

		vv, ok := v.Type.(*ast.StructType)
		if !ok {
			continue
		}

		info.StructName = name
		for _, field := range vv.Fields.List {
			//*ast.FuncType *ast.Ident
			//TODO panic on FuncType
			err := t.parseField(name, field, &info.Members, true, false)
			if err != nil {
				return err
			}

			if info.TableName == "" {
				continue
			}

			if field.Comment == nil {
				continue
			}

			comment := field.Comment.List[0]
			indexText := comment.Text
			indexInfo := splitAndTrimSpace(comment.Text, ":")
			//parse comment like://primary:t.primary
			if len(indexInfo) <= 1 {
				continue
			}

			dbType := indexInfo[0]
			if dbType == "//primary" {
				if len(indexInfo) == 2 {
					primary := indexInfo[1]
					if primary == "" {
						return t.newError(comment.Pos(), "Empty primary key in struct "+name)
					}

					if info.PrimaryKey != "" {
						return t.newError(comment.Pos(), "Duplicated primary key in struct "+name)
					}
					info.PrimaryKey = primary
				} else {
					errMsg := fmt.Sprintf("Invalid primary key in struct %s: %s", name, indexText)
					return t.newError(comment.Pos(), errMsg)
				}
			} else if _, ok := t.indexTypeMap[dbType[2:]]; ok {
				if len(indexInfo) != 4 {
					errMsg := fmt.Sprintf("Invalid index DB in struct %s: %s", name, indexText)
					return t.newError(comment.Pos(), errMsg)
				}

				idx := dbType[2:]
				name := indexInfo[1]
				if name == "" {
					return t.newError(comment.Pos(), "Invalid name in: "+indexText)
				}

				for i := range info.SecondaryIndexes {
					if info.SecondaryIndexes[i].Name == name {
						errMsg := fmt.Sprintf("Duplicated index name %s in %s", name, indexText)
						return t.newError(comment.Pos(), errMsg)
					}
				}

				getter := indexInfo[2]
				if getter == "" {
					return t.newError(comment.Pos(), "Invalid getter in: "+indexText)
				}

				setter := indexInfo[3]
				if setter == "" {
					return t.newError(comment.Pos(), "Invalid setter in: "+indexText)
				}
				indexInfo := SecondaryIndexInfo{idx, name, getter, setter}
				info.SecondaryIndexes = append(info.SecondaryIndexes, indexInfo)
			}
		}
		t.structs = append(t.structs, info)
		if strings.TrimSpace(lastLineDoc) == "//packer" {
			t.abiStructsMap[info.StructName] = &t.structs[len(t.structs)-1]
		}
	}
	return nil
}

func IsNameValid(name string) bool {
	return NameToString(StringToName(name)) == name
}

func (t *CodeGenerator) getLineInfo(p token.Pos) string {
	pos := t.fset.Position(p)
	return pos.String()
}

func (t *CodeGenerator) parseFunc(f *ast.FuncDecl) error {
	if f.Name.Name == "main" {
		t.hasMainFunc = true
	} else if f.Name.Name == "NewContract" {
		t.hasNewContractFunc = true
	}

	if f.Doc == nil {
		return nil
	}
	n := len(f.Doc.List)
	doc := f.Doc.List[n-1]
	text := strings.TrimSpace(doc.Text)

	//	parts := Split(text)
	parts := strings.Fields(text)
	if len(parts) < 2 || len(parts) > 3 {
		return nil
	}

	if parts[0] == "//action" || parts[0] == "//notify" {
	} else {
		return nil
	}

	actionName := parts[1]
	if !IsNameValid(actionName) {
		errMsg := fmt.Sprintf("Invalid action name: %s", actionName)
		return t.newError(doc.Pos(), errMsg)
	}

	if _, ok := t.actionMap[actionName]; ok {
		errMsg := fmt.Sprintf("Dumplicated action name: %s", actionName)
		return t.newError(doc.Pos(), errMsg)
	}

	ignore := false
	if len(parts) == 3 {
		if parts[2] != "ignore" {
			errMsg := fmt.Sprintf("Bad action, %s not recognized as a valid paramater", parts[2])
			return errors.New(errMsg)
		}
		ignore = true
	}

	t.actionMap[actionName] = true

	action := ActionInfo{}
	action.ActionName = actionName
	action.FuncName = f.Name.Name
	action.Ignore = ignore

	if parts[0] == "//notify" {
		action.IsNotify = true
	} else {
		action.IsNotify = false
	}

	if f.Recv.List != nil {
		for _, v := range f.Recv.List {
			expr := v.Type.(*ast.StarExpr)
			ident := expr.X.(*ast.Ident)
			if ident.Obj != nil {
				obj := ident.Obj
				action.StructName = obj.Name
			}
		}
	}

	for _, v := range f.Type.Params.List {
		err := t.parseField(f.Name.Name, v, &action.Members, false, ignore)
		if err != nil {
			return err
		}
	}
	t.actions = append(t.actions, action)
	return nil
}

var largePackages = []string{
	"\"strconv\"",
	"\"fmt\"",
}

var errorPackages = []string{
	"\"log\"",
}

func isLargePackage(pkgName string) bool {
	for i := range largePackages {
		if pkgName == largePackages[i] {
			return true
		}
	}
	return false
}

func isErrorPackage(pkgName string) bool {
	for i := range errorPackages {
		if pkgName == errorPackages[i] {
			return true
		}
	}
	return false
}

func (t *CodeGenerator) ParseGoFile(goFile string) error {
	t.currentFile = goFile
	file, err := parser.ParseFile(t.fset, goFile, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	for _, imp := range file.Imports {
		pkgName := imp.Path.Value
		if isLargePackage(pkgName) {
			color.Set(color.FgYellow)
			log.Printf("WARNING: use of package %s will greatly increase Smart Contract size\n", pkgName)
			color.Unset()
		} else if isErrorPackage(pkgName) {
			color.Set(color.FgRed)
			fmt.Printf("ERROR: use of package %s is not allowed in Smart Contracts!\n", pkgName)
			color.Unset()
			os.Exit(-1)
		}
	}

	if file.Name.Name != "main" {
		return nil
	}

	log.Println("Processing file:", goFile)

	for _, decl := range file.Decls {
		switch v := decl.(type) {
		case *ast.FuncDecl:
			if err := t.parseFunc(v); err != nil {
				return err
			}
		case *ast.GenDecl:
			if err := t.parseStruct(file.Name.Name, v); err != nil {
				return err
			}
		default:
			return t.newError(decl.Pos(), "Unknown declaration")
		}
	}

	return nil
}

func (t *CodeGenerator) writeCode(format string, a ...interface{}) {
	fmt.Fprintf(t.codeFile, "\n")
	fmt.Fprintf(t.codeFile, format, a...)

	if format == "}" { //end of function
		fmt.Fprintf(t.codeFile, "\n")
	}
}

func (t *CodeGenerator) genActionCode(notify bool) error {
	t.writeCode("        switch action.N {")
	for _, action := range t.actions {
		if action.IsNotify == notify {
		} else {
			continue
		}
		t.writeCode("        case uint64(%d): //%s", StringToName(action.ActionName), action.ActionName)
		if !action.Ignore {
			t.writeCode("            t := %s{}", action.ActionName)
			t.writeCode("            t.Unpack(data)")
			args := "("
			for i, member := range action.Members {
				if member.LeadingType == TYPE_POINTER {
					args += "&t." + member.Name
				} else {
					args += "t." + member.Name
				}
				if i != len(action.Members)-1 {
					args += ", "
				}
			}
			args += ")"
			t.writeCode("            contract.%s%s", action.FuncName, args)
		} else {
			args := "("
			for i, member := range action.Members {
				log.Println("+++++gen action code:", member.Name, member.Type)
				if member.LeadingType == TYPE_POINTER || member.LeadingType == TYPE_SLICE {
					//args += "&t." + member.Name
					args += "nil"
					if i != len(action.Members)-1 {
						args += ", "
					}
				} else {
					return fmt.Errorf("ignore action has not pointer parameter: %s", member.Name)
				}
			}
			args += ")"
			t.writeCode("            contract.%s%s", action.FuncName, args)
		}
	}
	t.writeCode("        }")
	return nil
}

func (t *CodeGenerator) GenActionCode() {
	t.genActionCode(false)
}

func (t *CodeGenerator) GenNotifyCode() {
	t.genActionCode(true)
}

func (t *CodeGenerator) packNotArrayType(goName string, goType string, indent string) {
	// bytes
	var format string
	switch goType {
	case "bool":
		format = "enc.PackBool(t.%s)"
	case "int8":
		format = "enc.PackUint8(uint8(t.%s))"
	case "uint8":
		format = "enc.PackUint8(t.%s)"
	case "int16":
		format = "enc.PackInt16(t.%s)"
	case "uint16":
		format = "enc.PackUint16(t.%s)"
	case "int32":
		format = "enc.PackInt32(t.%s)"
	case "uint32":
		format = "enc.PackUint32(t.%s)"
	case "int64":
		format = "enc.PackInt64(t.%s)"
	case "uint64":
		format = "enc.PackUint64(t.%s)"
	case "chain.Int128":
		format = "enc.WriteBytes(t.%s[:])"
	case "chain.Uint128":
		format = "enc.WriteBytes(t.%s[:])"
	case "chain.VarInt32":
		format = "enc.PackVarInt32(int32(t.%s))"
	case "chain.VarUint32":
		format = "enc.PackVarUint32(uint32(t.%s))"
	case "float32":
		format = "enc.PackFloat32(t.%s)"
	case "float64":
		format = "enc.PackFloat64(t.%s)"
	case "float128":
		format = "enc.WriteBytes(t.%s[:])"
	case "bytes":
		format = "enc.PackBytes(t.%s)"
	case "string":
		format = "enc.PackString(t.%s)"
	case "chain.Name":
		format = "enc.PackUint64(t.%s.N)"
	case "chain.TimePoint", "chain.TimePointSec",
		"chain.BlockTimestampType", "chain.Checksum160",
		"chain.Checksum256", "chain.Checksum512",
		"chain.PublicKeyType", "chain.Signature",
		"chain.Symbol", "chain.SymbolCode",
		"chain.Asset", "chain.ExtendedAsset":
		format = "enc.Pack(&t.%s)"
	default:
		format = "enc.Pack(&t.%s)"
	}
	t.writeCode(indent+format, goName)
}

func (t *CodeGenerator) packArrayType(goName string, goType string) {
	if goType == "byte" {
		t.writeCode("    enc.PackBytes(t.%s)", goName)
	} else {
		t.writeCode("{")
		t.writeCode("    enc.PackLength(len(t.%[1]s))", goName)
		t.writeCode("    for i := range t.%[1]s {", goName)
		t.packNotArrayType(goName+"[i]", goType, "        ")
		t.writeCode("    }")
		t.writeCode("}")
	}
}

func (t *CodeGenerator) packType(member StructMember) {
	if member.Name == "" {
		log.Printf("anonymount Type does not supported currently: %s", member.Type)
		return
	}
	if member.LeadingType == TYPE_SLICE {
		t.packArrayType(member.Name, member.Type)
	} else if member.LeadingType == TYPE_ARRAY {
		t.writeCode("    for i := range t.%s {", member.Name)
		t.packNotArrayType(member.Name+"[i]", member.Type, "        ")
		t.writeCode("    }")
	} else {
		t.packNotArrayType(member.Name, member.Type, "    ")
	}
}

func (t *CodeGenerator) unpackType(funcName string, varName string) {
	t.writeCode("    %s = dec.%s()", varName, funcName)
}

func (t *CodeGenerator) unpackI(varName string) {
	t.writeCode("    dec.UnpackI(&%s)", varName)
}

func (t *CodeGenerator) unpackBaseType(varName string, typ string) {
	switch typ {
	case "bool":
		t.unpackType("UnpackBool", varName)
	case "byte":
		t.unpackType("UnpackUint8", varName)
	case "int8":
		t.unpackType("UnpackInt8", varName)
	case "uint8":
		t.unpackType("UnpackUint8", varName)
	case "int16":
		t.unpackType("UnpackInt16", varName)
	case "uint16":
		t.unpackType("UnpackUint16", varName)
	case "int32":
		t.unpackType("UnpackInt32", varName)
	case "uint32":
		t.unpackType("UnpackUint32", varName)
	case "int64":
		t.unpackType("UnpackInt64", varName)
	case "uint64":
		t.unpackType("UnpackUint64", varName)
	case "chain.Name":
		t.unpackType("UnpackName", varName)
	case "bytes":
		t.unpackType("UnpackBytes", varName)
	case "string":
		t.unpackType("UnpackString", varName)
	case "float32":
		t.unpackType("UnpackFloat32", varName)
	case "float64":
		t.unpackType("UnpackFloat64", varName)
	case "[]byte":
		t.unpackType("UnpackBytes", varName)
	//TODO: handle other types that does not have Unpacker interface
	case "int":
		panic("int type is not supported")
	default:
		t.unpackI(varName)
	}
	// int128
	// uint128
	// varint32
	// varuint32
	// float32
	// float64
	// float128
	// time_point
	// time_point_sec
	// block_timestamp_type
	// checksum160
	// checksum256
	// checksum512
	// public_key
	// signature
	// symbol
	// symbol_code
	// asset
	// extended_asset
}

func (t *CodeGenerator) unpackMember(member StructMember) {
	if member.Name == "" {
		log.Printf("anonymount Type does not supported currently: %s", member.Type)
		return
	}
	if member.LeadingType == TYPE_SLICE {
		t.writeCode("{")
		t.writeCode("    length := dec.UnpackLength()")
		t.writeCode("    t.%s = make([]%s, length)", member.Name, member.Type)
		t.writeCode("    for i:=0; i<length; i++ {")
		t.unpackBaseType(fmt.Sprintf("t.%s[i]", member.Name), member.Type)
		t.writeCode("    }")
		t.writeCode("}")
	} else {
		t.unpackBaseType("t."+member.Name, member.Type)
	}
}

func (t *CodeGenerator) genStruct(structName string, members []StructMember) {
	log.Println("+++action", structName)
	t.writeCode("type %s struct {", structName)
	for _, member := range members {
		if member.LeadingType == TYPE_SLICE {
			t.writeCode("    %s []%s", member.Name, member.Type)
		} else {
			t.writeCode("    %s %s", member.Name, member.Type)
		}
	}
	t.writeCode("}")
}

func (t *CodeGenerator) genPackCode(structName string, members []StructMember) {
	t.writeCode("func (t *%s) Pack() []byte {", structName)
	t.writeCode("    enc := chain.NewEncoder(t.Size())")
	for _, member := range members {
		t.packType(member)
	}
	t.writeCode("    return enc.GetBytes()\n}\n")
}

func (t *CodeGenerator) genUnpackCode(structName string, members []StructMember) {
	t.writeCode("func (t *%s) Unpack(data []byte) int {", structName)
	t.writeCode("    dec := chain.NewDecoder(data)")
	for _, member := range members {
		t.unpackMember(member)
	}
	t.writeCode("    return dec.Pos()\n}\n")
}

func (t *CodeGenerator) genPackCodeForSpecialStruct(specialType int, structName string, member StructMember) {
	if specialType == BinaryExtensionType {
		t.writeCode(`
func (t *%s) Pack() []byte {
	if !t.HasValue {
		return []byte{}
	}`, structName)
		t.writeCode("    enc := chain.NewEncoder(t.Size())")
		t.packType(member)
		t.writeCode("    return enc.GetBytes()\n}\n")
	} else if specialType == OptionalType {
		t.writeCode(`
func (t *%s) Pack() []byte {
	if !t.IsValid {
		return []byte{0}
	}`, structName)
		t.writeCode("    enc := chain.NewEncoder(t.Size())")
		t.writeCode("    enc.WriteUint8(uint8(1))")
		t.packType(member)
		t.writeCode("    return enc.GetBytes()\n}\n")
	}
}

func (t *CodeGenerator) genUnpackCodeForSpecialStruct(specialType int, structName string, member StructMember) {
	if specialType == BinaryExtensionType {
		t.writeCode(`
func (t *%s) Unpack(data []byte) int {
	if len(data) == 0 {
		t.HasValue = false
		return 0
	} else {
		t.HasValue = true
	}`, structName)
		t.writeCode("    dec := chain.NewDecoder(data)")
		t.unpackMember(member)
		t.writeCode("    return dec.Pos()\n}\n")
	} else if specialType == OptionalType {
		t.writeCode(`
func (t *%s) Unpack(data []byte) int {
	chain.Check(len(data) >= 1, "invalid data size")
    dec := chain.NewDecoder(data)
	valid := dec.ReadUint8()
	if valid == 0 {
		t.IsValid = false
		return 1
	} else if valid == 1 {
		t.IsValid = true
	} else {
		chain.Check(false, "invalid optional value")
	}`, structName)
		t.unpackMember(member)
		t.writeCode("    return dec.Pos()\n}\n")
	}
}

func (t *CodeGenerator) genSizeCodeForSpecialStruct(specialType int, structName string, member StructMember) {
	if specialType == BinaryExtensionType {
		t.writeCode("func (t *%s) Size() int {", structName)
		t.writeCode("    size := 0")
		t.writeCode("    if !t.HasValue {")
		t.writeCode("        return size")
		t.writeCode("    }")
		if member.LeadingType == TYPE_SLICE {
			t.writeCode("    size += chain.PackedVarUint32Length(uint32(len(t.%s)))", member.Name)
			t.calcArrayMemberSize(member.Name, member.Type)
		} else if member.LeadingType == TYPE_ARRAY {
			t.writeCode("    for i := range t.%s {", member.Name)
			t.calcNotArrayMemberSize(member.Name+"[i]", member.Type)
			t.writeCode("    }")
		} else {
			t.calcNotArrayMemberSize(member.Name, member.Type)
		}
		t.writeCode("    return size")
		t.writeCode("}")
	} else if specialType == OptionalType {
		t.writeCode("func (t *%s) Size() int {", structName)
		t.writeCode("    size := 1")
		t.writeCode("    if !t.IsValid {")
		t.writeCode("        return size")
		t.writeCode("    }")
		if member.LeadingType == TYPE_SLICE {
			t.writeCode("    size += chain.PackedVarUint32Length(uint32(len(t.%s)))", member.Name)
			t.calcArrayMemberSize(member.Name, member.Type)
		} else {
			t.calcNotArrayMemberSize(member.Name, member.Type)
		}
		t.writeCode("    return size")
		t.writeCode("}")
	}
}

func (t *CodeGenerator) calcNotArrayMemberSize(name string, goType string) {
	var code string

	switch goType {
	case "string":
		code = fmt.Sprintf("    size += chain.PackedVarUint32Length(uint32(len(t.%s))) + len(t.%s)", name, name)
	case "byte":
		code = "    size += 1"
	case "bool":
		code = "    size += 1"
	case "int8":
		code = "    size += 1"
	case "uint8":
		code = "    size += 1"
	case "int16":
		code = "    size += 2"
	case "uint16":
		code = "    size += 2"
	case "int":
		code = "    size += 4"
	case "int32":
		code = "    size += 4"
	case "uint32":
		code = "    size += 4"
	case "int64":
		code = "    size += 8"
	case "uint64":
		code = "    size += 8"
	case "chain.Int128":
		code = "    size += 16"
	case "chain.Uint128":
		code = "    size += 16"
	case "chain.Uint256":
		code = "    size += 32"
	case "float32":
		code = "    size += 4"
	case "float64":
		code = "    size += 8"
	case "chain.Name":
		code = "    size += 8"
	case "chain.Signature":
		code = fmt.Sprintf("    size += t.%s.Size()", name)
	case "chain.PublicKey":
		code = fmt.Sprintf("    size += t.%s.Size()", name)
	case "chain.Symbol":
		code = "    size += 8"
	default:
		code = fmt.Sprintf("	size += t.%[1]s.Size()", name)
	}
	t.writeCode(code + " //" + name)
}

func (t *CodeGenerator) calcArrayMemberSize(name string, goType string) {
	switch goType {
	case "byte":
		t.writeCode("    size += len(t.%s)", name)
	case "[]byte":
		t.writeCode(`	for i := range t.%[1]s {
	size += chain.PackedVarUint32Length(uint32(len(t.%[1]s[i]))) + len(t.%[1]s[i])
}`, name)
	case "string":
		t.writeCode(`    for i := range t.%[1]s {
	 size += chain.PackedVarUint32Length(uint32(len(t.%[1]s[i]))) + len(t.%[1]s[i])
}`, name)
	case "bool":
		t.writeCode("    size += len(t.%s)", name)
	case "int8":
		t.writeCode("    size += len(t.%s)", name)
	case "uint8":
		t.writeCode("    size += len(t.%s)", name)
	case "int16":
		t.writeCode("    size += len(t.%s)*2", name)
	case "uint16":
		t.writeCode("    size += len(t.%s)*2", name)
	case "int":
		t.writeCode("    size += len(t.%s)*4", name)
	case "int32":
		t.writeCode("    size += len(t.%s)*4", name)
	case "uint32":
		t.writeCode("    size += len(t.%s)*4", name)
	case "int64":
		t.writeCode("    size += len(t.%s)*8", name)
	case "uint64":
		t.writeCode("    size += len(t.%s)*8", name)
	case "chain.Uint128":
		t.writeCode("    size += len(t.%s)*16", name)
	case "chain.Uint256":
		t.writeCode("    size += len(t.%s)*32", name)
	case "float32":
		t.writeCode("    size += len(t.%s)*4", name)
	case "float64":
		t.writeCode("    size += len(t.%s)*8", name)
	case "chain.Name":
		t.writeCode("    size += len(t.%s)*8", name)
	default:
		t.writeCode(`
    for i := range t.%[1]s {
        size += t.%[1]s[i].Size()
    }`, name)
	}
}

func (t *CodeGenerator) genSizeCode(structName string, members []StructMember) {
	t.writeCode("func (t *%s) Size() int {", structName)
	t.writeCode("    size := 0")
	for _, member := range members {
		if member.LeadingType == TYPE_SLICE {
			t.writeCode("    size += chain.PackedVarUint32Length(uint32(len(t.%s)))", member.Name)
			t.calcArrayMemberSize(member.Name, member.Type)
		} else if member.LeadingType == TYPE_ARRAY {
			t.writeCode("    for i := range t.%s {", member.Name)
			t.calcNotArrayMemberSize(member.Name+"[i]", member.Type)
			t.writeCode("    }")
		} else {
			t.calcNotArrayMemberSize(member.Name, member.Type)
		}
	}
	t.writeCode("    return size")
	t.writeCode("}")
}

func GetIndexType(index string) string {
	switch index {
	case "IDX64":
		return "uint64"
	case "IDX128":
		return "chain.Uint128"
	case "IDX256":
		return "chain.Uint256"
	case "IDXFloat64":
		return "float64"
	case "IDXFloat128":
		return "chain.Float128"
	default:
		panic(fmt.Sprintf("unknown secondary index type: %s", index))
	}
}

func indexTypeToSecondaryType(indexType string) string {
	switch indexType {
	case "IDX64":
		return "uint64"
	case "IDX128":
		return "chain.Uint128"
	case "IDX256":
		return "chain.Uint256"
	case "IDXFloat64":
		return "float64"
	case "IDXFloat128":
		return "chain.Float128"
	default:
		panic(fmt.Sprintf("unknown secondary index type: %s", indexType))
	}
	return ""
}

func indexTypeToSecondaryDBName(indexType string) string {
	switch indexType {
	case "IDX64":
		return "IdxDB64"
	case "IDX128":
		return "IdxDB128"
	case "IDX256":
		return "IdxDB256"
	case "IDXFloat64":
		return "IdxDBFloat64"
	case "IDXFloat128":
		return "IdxDBFloat128"
	default:
		panic(fmt.Sprintf("unknown secondary index type: %s", indexType))
	}
	return ""
}

func (t *CodeGenerator) GenCode() error {
	f, err := os.Create(t.dirName + "/generated.go")
	if err != nil {
		return err
	}
	t.codeFile = f

	for _, info := range t.structs {
		log.Println("++struct:", info.StructName)
	}

	t.writeCode(cImportCode)

	for _, action := range t.actions {
		t.genStruct(action.ActionName, action.Members)
		t.genPackCode(action.ActionName, action.Members)
		t.genUnpackCode(action.ActionName, action.Members)
		t.genSizeCode(action.ActionName, action.Members)
	}

	for _, _struct := range t.abiStructsMap {
		t.genPackCode(_struct.StructName, _struct.Members)
		t.genUnpackCode(_struct.StructName, _struct.Members)
		t.genSizeCode(_struct.StructName, _struct.Members)
	}

	// for i := range t.structs {
	// 	_struct := &t.structs[i]
	// 	if _, ok := t.abiStructsMap[_struct.StructName]; ok {
	// 		continue
	// 	}
	// 	t.genPackCode(_struct.StructName, _struct.Members)
	// 	t.genUnpackCode(_struct.StructName, _struct.Members)
	// 	t.genSizeCode(_struct.StructName, _struct.Members)
	// }

	for i := range t.structs {
		table := &t.structs[i]
		if table.TableName == "" {
			continue
		}

		t.writeCode(`
func %sDBNameToIndex(indexName string) int {
	switch indexName {`, table.StructName)

		for i, index := range table.SecondaryIndexes {
			if index.Name != "" {
				t.writeCode(`	case "%s":`, index.Name)
				t.writeCode(`	    return %d`, i)
			}
		}

		t.writeCode(`	default:
		panic("unknow indexName")
	}
}`)

		t.writeCode("var (\n	%sSecondaryTypes = []int{", table.StructName)
		for _, index := range table.SecondaryIndexes {
			t.writeCode("        database.%s,", index.Type)
		}
		t.writeCode("})")

		t.writeCode(`
func (t *%s) GetSecondaryValue(index int) interface{} {
	switch index {`, table.StructName)

		for i, index := range table.SecondaryIndexes {
			t.writeCode(`    case %d:
		return %s`, i, index.Getter)

		}
		t.writeCode(`	default:
		panic("index out of bound")
	}
}`)

		t.writeCode(`
func (t *%s) SetSecondaryValue(index int, v interface{}) {
	switch index {`, table.StructName)
		for i, index := range table.SecondaryIndexes {
			t.writeCode(`    case %d:`, i)
			value := fmt.Sprintf("v.(%s)", GetIndexType(index.Type))
			if strings.Index(index.Setter, "%v") >= 0 {
				t.writeCode(fmt.Sprintf("        "+index.Setter, value))
			} else {
				t.writeCode(fmt.Sprintf("        %s=%s", index.Setter, value))
			}
		}

		t.writeCode(`	default:
			panic("unknown index")
		}
}`)

		if table.PrimaryKey != "" {
			t.writeCode("func (t *%s) GetPrimary() uint64 {", table.StructName)
			t.writeCode("    return %s", table.PrimaryKey)
			t.writeCode("}")
		}

		t.writeCode(cUnpackerCode, table.StructName)

		//generate singleton db code
		if table.Singleton {
			t.writeCode(cSingletonCode, table.StructName, StringToName(table.TableName))
			continue
		}

		t.writeCode(cDBTemplate, table.StructName, StringToName(table.TableName), table.TableName)
		t.writeCode(cNewMultiIndexTemplate, table.StructName, StringToName(table.TableName), table.TableName)
		for i, index := range table.SecondaryIndexes {
			idxTable := (StringToName(table.TableName) & uint64(0xfffffffffffffff0)) | uint64(i)
			if index.Type == "IDX64" {
				t.writeCode("    mi.IDXDBs[%[1]d] = database.NewIdxDB64(%[1]d, code.N, scope.N, uint64(%[2]d))", i, idxTable)
			} else if index.Type == "IDX128" {
				t.writeCode("    mi.IDXDBs[%[1]d] = database.NewIdxDB128(%[1]d, code.N, scope.N, uint64(%[2]d))", i, idxTable)
			} else if index.Type == "IDX256" {
				t.writeCode("    mi.IDXDBs[%[1]d] = database.NewIdxDB256(%[1]d, code.N, scope.N, uint64(%[2]d))", i, idxTable)
			} else if index.Type == "IDXFloat64" {
				t.writeCode("    mi.IDXDBs[%[1]d] = database.NewIdxDBFloat64(%[1]d, code.N, scope.N, uint64(%[2]d))", i, idxTable)
			} else if index.Type == "IDXFloat128" {
				t.writeCode("    mi.IDXDBs[%[1]d] = database.NewIdxDBFloat128(%[1]d, code.N, scope.N, uint64(%[2]d))", i, idxTable)
			}
		}
		t.writeCode("    return &%[1]sDB{mi}\n}", table.StructName)

		for i := range table.SecondaryIndexes {
			index := &table.SecondaryIndexes[i]
			t.writeCode(cGetDBTemplate, table.StructName, index.Name, i, indexTypeToSecondaryDBName(index.Type))
		}
	}

	for i := range t.specialAbiTypes {
		ext := &t.specialAbiTypes[i]
		t.genPackCodeForSpecialStruct(ext.typ, ext.name, ext.member)
		t.genUnpackCodeForSpecialStruct(ext.typ, ext.name, ext.member)
		t.genSizeCodeForSpecialStruct(ext.typ, ext.name, ext.member)
	}

	t.writeCode(cDummyCode)

	if t.hasMainFunc {
		return nil
	}

	t.writeCode(cMainCode)

	t.writeCode("    if receiver == firstReceiver {")
	t.GenActionCode()
	t.writeCode("    }")

	t.writeCode("    if receiver != firstReceiver {")
	t.GenNotifyCode()
	t.writeCode("    }")
	t.writeCode("}")
	return nil
}

func (t *CodeGenerator) GenAbi() error {
	var abiFile string
	if t.contractName == "" {
		abiFile = t.dirName + "/generated.abi"
	} else {
		abiFile = t.dirName + "/" + t.contractName + ".abi"
	}

	f, err := os.Create(abiFile)
	if err != nil {
		panic(err)
	}

	abi := ABI{}
	abi.Version = "eosio::abi/1.1"
	abi.Structs = make([]ABIStruct, 0, len(t.structs)+len(t.actions))

	abi.Types = []string{}
	abi.Actions = []ABIAction{}
	abi.Tables = []ABITable{}
	abi.RicardianClauses = []string{}
	abi.Variants = []string{}
	abi.AbiExtensions = []string{}
	abi.ErrorMessages = []string{}

	for _, _struct := range t.abiStructsMap {
		s := ABIStruct{}
		s.Name = _struct.StructName
		s.Base = ""
		s.Fields = make([]ABIStructField, 0, len(_struct.Members))
		for _, member := range _struct.Members {
			abiType, err := t.convertType(member)
			if err != nil {
				return err
			}
			field := ABIStructField{Name: member.Name, Type: abiType}
			s.Fields = append(s.Fields, field)
		}
		abi.Structs = append(abi.Structs, s)
	}

	for _, action := range t.actions {
		s := ABIStruct{}
		s.Name = action.ActionName
		s.Base = ""
		s.Fields = make([]ABIStructField, 0, len(action.Members))
		for _, member := range action.Members {
			abiType, err := t.convertType(member)
			if err != nil {
				return err
			}
			field := ABIStructField{Name: member.Name, Type: abiType}
			s.Fields = append(s.Fields, field)
		}
		abi.Structs = append(abi.Structs, s)
	}

	abi.Actions = make([]ABIAction, 0, len(t.actions))
	for _, action := range t.actions {
		a := ABIAction{}
		a.Name = action.ActionName
		a.Type = action.ActionName
		a.RicardianContract = ""
		abi.Actions = append(abi.Actions, a)
	}

	for _, table := range t.structs {
		if table.TableName == "" {
			continue
		}
		abiTable := ABITable{}
		abiTable.Name = table.TableName
		abiTable.Type = table.StructName
		abiTable.IndexType = "i64"
		abiTable.KeyNames = []string{}
		abiTable.KeyTypes = []string{}
		abi.Tables = append(abi.Tables, abiTable)
	}

	sort.Slice(abi.Structs, func(i, j int) bool {
		return strings.Compare(abi.Structs[i].Name, abi.Structs[j].Name) < 0
	})

	sort.Slice(abi.Types, func(i, j int) bool {
		return strings.Compare(abi.Types[i], abi.Types[j]) < 0
	})

	sort.Slice(abi.Actions, func(i, j int) bool {
		return strings.Compare(abi.Actions[i].Name, abi.Actions[j].Name) < 0
	})

	sort.Slice(abi.Tables, func(i, j int) bool {
		return strings.Compare(abi.Tables[i].Name, abi.Tables[j].Name) < 0
	})

	// Structs          []ABIStruct `json:"structs"`
	// Types            []string    `json:"types"`
	// Actions          []ABIAction `json:"actions"`
	// Tables           []ABITable  `json:"tables"`

	result, err := json.MarshalIndent(abi, "", "    ")
	if err != nil {
		panic(err)
	}
	f.Write(result)
	f.Close()
	return nil
}

func (t *CodeGenerator) FetchAllGoFiles(dir string) []string {
	goFiles := []string{}
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		if filepath.Ext(f.Name()) != ".go" {
			continue
		}

		if f.Name() == "generated.go" {
			continue
		}
		goFiles = append(goFiles, f.Name())
	}
	return goFiles
}

func (t *CodeGenerator) Finish() {
	t.codeFile.Close()
}

func (t *CodeGenerator) isSpecialAbiType(goType string) (string, bool) {
	for _, specialType := range t.specialAbiTypes {
		if specialType.name == goType {
			return specialType.member.Type, true
		}
	}
	return "", false
}

func (t *CodeGenerator) addAbiStruct(s *StructInfo) {
	t.abiStructsMap[s.StructName] = s
	for _, member := range s.Members {
		s2, ok := t.structMap[member.Type]
		if ok {
			t.addAbiStruct(s2)
			continue
		}

		typeName, ok := t.isSpecialAbiType(member.Type)
		if ok {
			log.Println("++++++++++isSpecialAbiType:", typeName)
			s2, ok := t.structMap[typeName]
			if ok {
				t.addAbiStruct(s2)
			}
		}
	}
}

func (t *CodeGenerator) Analyse() {
	for i := range t.structs {
		s := &t.structs[i]
		t.structMap[s.StructName] = s
	}

	for _, action := range t.actions {
		for _, member := range action.Members {
			item, ok := t.structMap[member.Type]
			if ok {
				t.addAbiStruct(item)
			}
		}
	}

	for _, item := range t.structs {
		if item.TableName == "" {
			continue
		}

		item2, ok := t.structMap[item.StructName]
		if ok {
			t.addAbiStruct(item2)
		}
	}
}

func GenerateCode(inFile string) error {
	// log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))

	gen := NewCodeGenerator()
	gen.fset = token.NewFileSet()

	if filepath.Ext(inFile) == ".go" {
		gen.dirName = filepath.Dir(inFile)
		if err := gen.ParseGoFile(inFile); err != nil {
			return err
		}
	} else {
		gen.dirName = inFile
		goFiles := gen.FetchAllGoFiles(inFile)
		for _, f := range goFiles {
			if err := gen.ParseGoFile(f); err != nil {
				return err
			}
		}
	}

	if gen.contractStructName != "" {
		if !gen.hasNewContractFunc {
			errorMsg := `NewContract function not defined, Please define it like this: func NewContract(receiver, firstReceiver, action chain.Name) *` + gen.contractStructName
			return errors.New(errorMsg)
		}
	}

	gen.Analyse()
	if err := gen.GenAbi(); err != nil {
		return err
	}

	if err := gen.GenCode(); err != nil {
		return err
	}
	gen.Finish()
	return nil
}