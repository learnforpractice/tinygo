package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
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

func Split(s string) []string {
	aa := strings.Split(s, " ")
	ret := []string{}
	for i := range aa {
		s := strings.TrimSpace(aa[i])
		if s != "" {
			ret = append(ret, s)
		}
	}
	return ret
}

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

type MemberType struct {
	Name        string
	Type        string
	LeadingType int
}

type ActionInfo struct {
	ActionName string
	FuncName   string
	StructName string
	Members    []MemberType
	IsNotify   bool
	Ignore     bool
}

type SecondaryIndexInfo struct {
	Type   string
	Name   string
	Getter string
	Setter string
}

type StructInfo struct {
	PackageName      string
	TableName        string
	Singleton        bool
	StructName       string
	Comment          string
	PrimaryKey       string
	SecondaryIndexes []SecondaryIndexInfo
	Members          []MemberType
}

type CodeGenerator struct {
	dirName      string
	currentFile  string
	contractName string
	fset         *token.FileSet
	codeFile     *os.File
	actions      []ActionInfo
	structs      []StructInfo
	structMap    map[string]*StructInfo

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

func (t *CodeGenerator) convertToAbiType(goType string) (string, error) {
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
	return "", fmt.Errorf(msg)
}

func (t *CodeGenerator) convertType(goType MemberType) (string, error) {
	//special case for []byte type
	if goType.Type == "byte" && goType.LeadingType == TYPE_SLICE {
		return "bytes", nil
	}

	abiType, err := t.convertToAbiType(goType.Type)
	if err != nil {
		return "", err
	}

	if goType.LeadingType == TYPE_SLICE {
		if abiType == "byte" {
			return "bytes", nil
		}
		return abiType + "[]", nil
	}
	return abiType, nil
}

func (t *CodeGenerator) newError(p token.Pos, msg string) error {
	errMsg := fmt.Sprintf("%s\n%s", t.getLineInfo(p), msg)
	return errors.New(errMsg)
}

func (t *CodeGenerator) parseField(structName string, field *ast.Field, memberList *[]MemberType, isStructField bool, ignore bool) error {
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

	log.Printf("%v %v %v\n", field.Names, field.Comment, field.Doc)

	switch fieldType := field.Type.(type) {
	case *ast.Ident:
		if field.Names != nil {
			for _, v := range field.Names {
				member := MemberType{}
				member.Name = v.Name
				member.Type = fieldType.Name
				*memberList = append(*memberList, member)
			}
		} else {
			//TODO: parse anonymous struct
			member := MemberType{}
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
				member := MemberType{}
				member.Name = name.Name
				member.Type = v.Name
				member.LeadingType = leadingType
				*memberList = append(*memberList, member)
			}
		case *ast.ArrayType:
			for _, name := range field.Names {
				if ident, ok := v.Elt.(*ast.Ident); ok {
					member := MemberType{}
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
				member := MemberType{}
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
			member := MemberType{}
			member.Name = name.Name
			member.Type = ident.Name + "." + fieldType.Sel.Name
			// member.IsArray = false
			*memberList = append(*memberList, member)
		}
	case *ast.StarExpr:
		//Do not parse pointer type in struct
		if isStructField {
			log.Printf("+++++++Pointer type %v in %s ignored\n", field.Names, structName)
			return nil
		}

		switch v2 := fieldType.X.(type) {
		case *ast.Ident:
			for _, name := range field.Names {
				member := MemberType{}
				member.Name = name.Name
				member.Type = v2.Name
				member.LeadingType = TYPE_POINTER
				*memberList = append(*memberList, member)
			}
		case *ast.SelectorExpr:
			switch x := v2.X.(type) {
			case *ast.Ident:
				for _, name := range field.Names {
					member := MemberType{}
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
		errMsg := fmt.Sprintf("Unsupported field: %v", field.Names)
		return t.newError(field.Pos(), errMsg)
	}
	return nil
}

func (t *CodeGenerator) parseStruct(packageName string, v *ast.GenDecl) error {
	if v.Tok != token.TYPE {
		return nil
	}
	info := StructInfo{}
	info.PackageName = packageName
	isContractStruct := false
	if v.Doc != nil {
		n := len(v.Doc.List)
		doc := v.Doc.List[n-1]
		text := strings.TrimSpace(doc.Text)
		if strings.HasPrefix(text, "//table") {
			items := Split(text)
			if len(items) == 2 && (items[0] == "//table") {
				tableName := items[1]
				if !IsNameValid(tableName) {
					return t.newError(doc.Pos(), "Invalid table name:"+tableName)
				}
				info.TableName = items[1]
			} else if (len(items) == 3) && (items[0] == "//table") {
				tableName := items[1]
				if !IsNameValid(tableName) {
					return t.newError(doc.Pos(), "Invalid table name:"+tableName)
				}
				info.TableName = items[1]
				if items[2] == "singleton" {
					info.Singleton = true
				}
			}
		} else if strings.HasPrefix(text, "//contract") {
			items := Split(text)
			if len(items) == 2 {
				name := items[1]
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
			log.Println("++++++field.Names", name, field.Names, field.Type)
			//*ast.FuncType *ast.Ident
			//TODO panic on FuncType
			if info.TableName != "" && field.Comment != nil {
				comment := field.Comment.List[0]
				indexText := strings.TrimSpace(comment.Text)
				indexInfo := strings.Split(indexText, ":")
				//parse comment like://primary:t.primary
				if len(indexInfo) > 1 {
					dbType := strings.TrimSpace(indexInfo[0])
					if dbType == "//primary" {
						if len(indexInfo) == 2 {
							primary := strings.TrimSpace(indexInfo[1])
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
						name := strings.TrimSpace(indexInfo[1])
						if name == "" {
							return t.newError(comment.Pos(), "Invalid name in: "+indexText)
						}

						for i := range info.SecondaryIndexes {
							if info.SecondaryIndexes[i].Name == name {
								errMsg := fmt.Sprintf("Duplicated index name %s in %s", name, indexText)
								return t.newError(comment.Pos(), errMsg)
							}
						}

						getter := strings.TrimSpace(indexInfo[2])
						if getter == "" {
							return t.newError(comment.Pos(), "Invalid getter in: "+indexText)
						}

						setter := strings.TrimSpace(indexInfo[3])
						if setter == "" {
							return t.newError(comment.Pos(), "Invalid setter in: "+indexText)
						}
						indexInfo := SecondaryIndexInfo{idx, name, getter, setter}
						info.SecondaryIndexes = append(info.SecondaryIndexes, indexInfo)
					}
				}
			}

			err := t.parseField(name, field, &info.Members, true, false)
			if err != nil {
				return err
			}
		}
		t.structs = append(t.structs, info)
	}
	return nil
}

func IsNameValid(name string) bool {
	return NameToString(StringToName(name)) == name
}

func (t *CodeGenerator) getLineInfo(p token.Pos) string {
	pos := t.fset.Position(p)
	return pos.String()
	// log.Println(pos.String())
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

	items := Split(text)
	if len(items) < 2 || len(items) > 3 {
		return nil
	}

	if items[0] == "//action" || items[0] == "//notify" {
	} else {
		return nil
	}

	actionName := items[1]
	if !IsNameValid(actionName) {
		errMsg := fmt.Sprintf("Invalid action name: %s", actionName)
		return t.newError(doc.Pos(), errMsg)
	}

	if _, ok := t.actionMap[actionName]; ok {
		errMsg := fmt.Sprintf("Dumplicated action name: %s", actionName)
		return t.newError(doc.Pos(), errMsg)
	}

	ignore := false
	if len(items) == 3 {
		if items[2] != "ignore" {
			errMsg := fmt.Sprintf("Bad action, %s not recognized as a valid paramater", items[2])
			return errors.New(errMsg)
		}
		ignore = true
	}

	t.actionMap[actionName] = true

	action := ActionInfo{}
	action.ActionName = actionName
	action.FuncName = f.Name.Name
	action.Ignore = ignore

	if items[0] == "//notify" {
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

func (t *CodeGenerator) ParseGoFile(goFile string) error {
	t.currentFile = goFile
	file, err := parser.ParseFile(t.fset, goFile, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	for _, c := range file.Comments {
		fmt.Printf("%v\n", c.Text())
	}

	if file.Name.Name != "main" {
		return nil
	}

	log.Println("parse file:", goFile)

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

func GenerateCode(inFile string) error {
	gen := NewCodeGenerator()
	gen.fset = token.NewFileSet()

	if filepath.Ext(inFile) == ".go" {
		gen.dirName = filepath.Dir(inFile)
		if err := gen.ParseGoFile(inFile); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	GenerateCode("test.go")

}
