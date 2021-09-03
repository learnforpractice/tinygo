package main

import (
	"encoding/json"
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

func _convertToAbiType(goType string) string {
	switch goType {
	case "int":
		return "int32"
	case "byte":
		return "uint8"
	case "chain.VarInt32":
		return "varint32"
	case "chain.VarUint32":
		return "varuint32"
	case "chain.Int128":
		return "int128"
	case "chain.Uint128":
		return "uint128"
	case "chain.Float128":
		return "float128"
	case "chain.Name":
		return "name"
	case "chain.TimePoint":
		return "time_point"
	case "chain.TimePointSec":
		return "time_point_sec"
	case "chain.BlockTimestampType":
		return "block_timestamp_type"
	case "chain.Checksum160":
		return "checksum160"
	case "chain.Checksum256":
		return "checksum256"
	case "chain.Uint256":
		return "checksum256"
	case "chain.Checksum512":
		return "checksum512"
	case "chain.PublicKey":
		return "public_key"
	case "chain.Signature":
		return "signature"
	case "chain.Symbol":
		return "symbol"
	case "chain.SymbolCode":
		return "symbol_code"
	case "chain.Asset":
		return "asset"
	case "chain.ExtendedAsset":
		return "extended_asset"
	default:
		return goType
	}
}

type MemberType struct {
	Name    string
	Type    string
	IsArray bool
}

type ActionInfo struct {
	ActionName string
	FuncName   string
	StructName string
	Members    []MemberType
	IsNotify   bool
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
	DirName            string
	contractName       string
	fset               *token.FileSet
	codeFile           *os.File
	Actions            []ActionInfo
	Structs            []StructInfo
	HasMainFunc        bool
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
	abiType := _convertToAbiType(goType)
	if _, ok := t.abiTypeMap[abiType]; ok {
		return abiType, nil
	}

	// check if type is a abi struct
	if _, ok := t.abiStructsMap[goType]; ok {
		return goType, nil
	}
	msg := fmt.Sprintf("type %s can not be converted to an ABI type", goType)
	if goType == "Asset" || goType == "Symbol" || goType == "Name" {
		msg += "\nDo you mean chain." + goType
	}
	return "", fmt.Errorf(msg)
}

func (t *CodeGenerator) convertType(goType MemberType) (string, error) {
	//special case for []byte type
	if goType.Type == "byte" && goType.IsArray {
		return "bytes", nil
	}

	abiType, err := t.convertToAbiType(goType.Type)
	if err != nil {
		return "", err
	}

	if goType.IsArray {
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
					log.Printf("contractName %s replace by %s", t.contractName, name)
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

			switch fieldType := field.Type.(type) {
			case *ast.Ident:
				if field.Names != nil {
					for _, v := range field.Names {
						member := MemberType{}
						member.Name = v.Name
						member.Type = fieldType.Name
						info.Members = append(info.Members, member)
					}
				} else {
					//TODO: parse anonymous struct
					member := MemberType{}
					member.Name = ""
					member.Type = fieldType.Name
					info.Members = append(info.Members, member)
				}
			case *ast.ArrayType:
				switch v := fieldType.Elt.(type) {
				case *ast.Ident:
					for _, name := range field.Names {
						member := MemberType{}
						member.Name = name.Name
						member.Type = v.Name
						member.IsArray = true
						info.Members = append(info.Members, member)
					}
				case *ast.ArrayType:
					for _, name := range field.Names {
						if ident, ok := v.Elt.(*ast.Ident); ok {
							member := MemberType{}
							member.Name = name.Name
							member.Type = "[]" + ident.Name
							member.IsArray = true
							info.Members = append(info.Members, member)
						} else {
							errMsg := fmt.Sprintf("Unsupported field %s in %s", name, info.StructName)
							return t.newError(field.Pos(), errMsg)
						}
					}
				case *ast.SelectorExpr:
					ident := v.X.(*ast.Ident)
					for _, name := range field.Names {
						member := MemberType{}
						member.Name = name.Name
						member.Type = ident.Name + "." + v.Sel.Name
						member.IsArray = true
						info.Members = append(info.Members, member)
					}
				default:
					errMsg := fmt.Sprintf("unsupported type: %T %s.%v", v, info.StructName, field.Names)
					return t.newError(field.Pos(), errMsg)
				}
			case *ast.SelectorExpr:
				ident := fieldType.X.(*ast.Ident)
				for _, name := range field.Names {
					member := MemberType{}
					member.Name = name.Name
					member.Type = ident.Name + "." + fieldType.Sel.Name
					member.IsArray = false
					info.Members = append(info.Members, member)
				}
			// case *ast.StarExpr:
			// 	s := fmt.Sprintf("++++++not supported type: %T %v\n", fieldType, fieldType)
			// 	log.Println(s)
			// 	if info.TableName != "" {
			// 		panic(s)
			// 	}
			// case *ast.FuncType:
			// 	s := fmt.Sprintf("++++++not supported type: %T %v\n", fieldType, fieldType)
			// 	log.Println(s)
			// 	if info.TableName != "" {
			// 		panic(s)
			// 	}
			//TODO parse anonymous struct
			// case *ast.StructType:
			// 	log.Printf("++++++anonymous struct does not supported currently: %s in %s", field.Names, name)
			// log.Printf("%T %v", fieldType, field.Names)
			default:
				errMsg := fmt.Sprintf("Unsupported field: %v in struct: %s", field.Names, name)
				return t.newError(field.Pos(), errMsg)
			}
		}
		t.Structs = append(t.Structs, info)
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
		t.HasMainFunc = true
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
	if len(items) != 2 {
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

	t.actionMap[actionName] = true

	action := ActionInfo{}
	action.ActionName = actionName
	action.FuncName = f.Name.Name

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
		switch expr := v.Type.(type) {
		case *ast.Ident:
			ident := expr
			for _, name := range v.Names {
				member := MemberType{}
				member.Name = name.Name
				member.Type = ident.Name
				action.Members = append(action.Members, member)
			}
		case *ast.ArrayType:
			ident := expr.Elt.(*ast.Ident)
			for _, name := range v.Names {
				member := MemberType{}
				member.Name = name.Name
				member.Type = ident.Name
				member.IsArray = true
				action.Members = append(action.Members, member)
			}
		case *ast.SelectorExpr:
			ident := expr.X.(*ast.Ident)
			for _, name := range v.Names {
				member := MemberType{}
				member.Name = name.Name
				member.Type = ident.Name + "." + expr.Sel.Name
				member.IsArray = false
				action.Members = append(action.Members, member)
			}
		default:
			panic("unknown type:" + fmt.Sprintf("%T", expr))
		}
	}
	t.Actions = append(t.Actions, action)
	return nil
}

func (t *CodeGenerator) ParseGoFile(goFile string) error {
	file, err := parser.ParseFile(t.fset, goFile, nil, parser.ParseComments)
	if err != nil {
		return err
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

func (t *CodeGenerator) writeCode(format string, a ...interface{}) {
	fmt.Fprintf(t.codeFile, "\n")
	fmt.Fprintf(t.codeFile, format, a...)
}

func (t *CodeGenerator) genActionCode(notify bool) {
	t.writeCode("        switch action.N {")
	for _, action := range t.Actions {
		if action.IsNotify == notify {
		} else {
			continue
		}
		t.writeCode("        case uint64(%d): //%s", StringToName(action.ActionName), action.ActionName)
		t.writeCode("            t := %s{}", action.ActionName)
		t.writeCode("            data := chain.ReadActionData()")
		t.writeCode("            t.Unpack(data)")
		args := "("
		for i, member := range action.Members {
			args += "t." + member.Name
			if i != len(action.Members)-1 {
				args += ", "
			}
		}
		args += ")"
		t.writeCode("            contract.%s%s", action.FuncName, args)
	}
	t.writeCode("        }")
}

func (t *CodeGenerator) GenActionCode() {
	t.genActionCode(false)
}

func (t *CodeGenerator) GenNotifyCode() {
	t.genActionCode(true)
}

func (t *CodeGenerator) packNotArrayType(goName string, goType string) {
	var format string
	switch goType {
	case "string":
		format = "    enc.PackString(t.%s)"
	case "bool":
		format = "    enc.PackBool(t.%s)"
	case "int8":
		format = "    enc.PackUint8(uint8(t.%s))"
	case "uint8":
		format = "    enc.PackUint8(t.%s)"
	case "int16":
		format = "    enc.PackInt16(t.%s)"
	case "uint16":
		format = "    enc.PackUint16(t.%s)"
	case "int":
		format = "    enc.PackInt32(int32(t.%s))"
	case "int32":
		format = "    enc.PackInt32(t.%s)"
	case "uint32":
		format = "    enc.PackUint32(t.%s)"
	case "int64":
		format = "    enc.PackInt64(t.%s)"
	case "uint64":
		format = "    enc.PackUint64(t.%s)"
	case "chain.Uint128":
		format = "    enc.WriteBytes(t.%s[:])"
	case "chain.Uint256":
		format = "    enc.WriteBytes(t.%s[:])"
	case "float32":
		format = "    enc.PackFloat32(t.%s)"
	case "float64":
		format = "    enc.PackFloat64(t.%s)"
	case "chain.Name":
		format = "    enc.PackUint64(t.%s.N)"
	default:
		format = "    enc.Pack(&t.%s)"
	}
	t.writeCode(format, goName)
}

func (t *CodeGenerator) packArrayType(goName string, goType string) {
	if goType == "byte" {
		t.writeCode("    enc.PackBytes(t.%s)", goName)
	} else {
		t.writeCode(`
{
	enc.PackLength(len(t.%[1]s))
	for _, v := range t.%[1]s {
		enc.Pack(&v)
	}
}`, goName)

	}
}

func (t *CodeGenerator) packType(member MemberType) {
	if member.Name == "" {
		log.Printf("anonymount Type does not supported currently: %s", member.Type)
		return
	}
	if member.IsArray {
		t.packArrayType(member.Name, member.Type)
	} else {
		t.packNotArrayType(member.Name, member.Type)
	}
}

func (t *CodeGenerator) unpackType(member MemberType) {
	if member.Name == "" {
		log.Printf("anonymount Type does not supported currently: %s", member.Type)
		return
	}
	if member.IsArray {
		t.writeCode("{")
		t.writeCode("    length, _ := dec.UnpackLength()")
		t.writeCode("    t.%s = make([]%s, length)", member.Name, member.Type)
		t.writeCode("    for i:=0; i<length; i++ {")
		t.writeCode("        dec.Unpack(&t.%s[i])", member.Name)
		t.writeCode("    }")
		t.writeCode("}")
	} else {
		t.writeCode("    dec.Unpack(&t.%s)", member.Name)
	}
}

func (t *CodeGenerator) genStruct(structName string, members []MemberType) {
	log.Println("+++action", structName)
	t.writeCode("type %s struct {", structName)
	for _, member := range members {
		if member.IsArray {
			t.writeCode("    %s []%s", member.Name, member.Type)
		} else {
			t.writeCode("    %s %s", member.Name, member.Type)
		}
	}
	t.writeCode("}\n")
}

func (t *CodeGenerator) genPackCode(structName string, members []MemberType) {
	t.writeCode("func (t *%s) Pack() []byte {", structName)
	t.writeCode("    enc := chain.NewEncoder(t.Size())")
	for _, member := range members {
		t.packType(member)
	}
	t.writeCode("    return enc.GetBytes()\n}\n")
}

func (t *CodeGenerator) genUnpackCode(structName string, members []MemberType) {
	t.writeCode("func (t *%s) Unpack(data []byte) (int, error) {", structName)
	t.writeCode("    dec := chain.NewDecoder(data)")
	for _, member := range members {
		t.unpackType(member)
	}
	t.writeCode("    return dec.Pos(), nil\n}\n")
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
	t.writeCode(code)
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

func (t *CodeGenerator) genSizeCode(structName string, members []MemberType) {
	t.writeCode("func (t *%s) Size() int {", structName)
	t.writeCode("    size := 0")
	for _, member := range members {
		if member.IsArray {
			t.writeCode("    size += chain.PackedVarUint32Length(uint32(len(t.%s)))", member.Name)
			t.calcArrayMemberSize(member.Name, member.Type)
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

func (t *CodeGenerator) GenCode() error {
	f, err := os.Create(t.DirName + "/generated.go")
	if err != nil {
		return err
	}
	t.codeFile = f

	for _, info := range t.Structs {
		log.Println("++struct:", info.StructName)
	}

	t.writeCode(`package main
import (
	"github.com/uuosio/chain"
    "github.com/uuosio/chain/database"
    "unsafe"
)
`)

	for _, action := range t.Actions {
		t.genStruct(action.ActionName, action.Members)
		t.genPackCode(action.ActionName, action.Members)
		t.genUnpackCode(action.ActionName, action.Members)
		t.genSizeCode(action.ActionName, action.Members)
	}

	for _, _struct := range t.Structs {
		t.genPackCode(_struct.StructName, _struct.Members)
		t.genUnpackCode(_struct.StructName, _struct.Members)
		t.genSizeCode(_struct.StructName, _struct.Members)
	}

	for i := range t.Structs {
		table := &t.Structs[i]
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

		t.writeCode(`
func %[1]sUnpacker(buf []byte) (database.MultiIndexValue, error) {
	v := &%[1]s{}
	_, err := v.Unpack(buf)
	if err != nil {
		return nil, err
	}
	return v, nil
}`, table.StructName)

		//generate singleton db code
		if table.Singleton {
			t.writeCode(`
func (d *%[1]s) GetPrimary() uint64 {
	return uint64(%[2]d)
}

type %[1]sDB struct {
	db *database.SingletonDB
}

func New%[1]sDB(code chain.Name, scope chain.Name) *%[1]sDB {
	table := chain.Name{N:uint64(%[2]d)}
	db := database.NewSingletonDB(code, scope, table, %[1]sUnpacker)
	return &%[1]sDB{db}
}

func (t *%[1]sDB) Set(data *%[1]s, payer chain.Name) {
	t.db.Set(data, payer)
}

func (t *%[1]sDB) Get() (*%[1]s) {
	data := t.db.Get()
	if data == nil {
		return nil
	}
	return data.(*%[1]s)
}
`, table.StructName, StringToName(table.TableName))
			continue
		}

		t.writeCode(`
type %[1]sDB struct {
	database.MultiIndexInterface
	I database.MultiIndexInterface
}

func New%[1]sDB(code chain.Name, scope chain.Name) *%[1]sDB {
	table := chain.Name{N:uint64(%[2]d)}
	db := database.NewMultiIndex(code, scope, table, %[1]sDBNameToIndex, %[1]sSecondaryTypes, %[1]sUnpacker)
	return &%[1]sDB{db, db}
}

func (mi *%[1]sDB) Store(v *%[1]s, payer chain.Name) {
	mi.I.Store(v, payer)
}

func (mi *%[1]sDB) Get(id uint64) (database.Iterator, *%[1]s) {
	it, data := mi.I.Get(id)
	if !it.IsOk() {
		return it, nil
	}
	return it, data.(*%[1]s)
}

func (mi *%[1]sDB) GetByIterator(it database.Iterator) (*%[1]s, error) {
	data, err := mi.I.GetByIterator(it)
	if err != nil {
		return nil, err
	}
	return data.(*%[1]s), nil
}

func (mi *%[1]sDB) Update(it database.Iterator, v *%[1]s, payer chain.Name) {
	mi.I.Update(it, v, payer)
}
`, table.StructName, StringToName(table.TableName))

	}

	t.writeCode(`
//eliminate unused package errors
func dummy() {
	if false {
		v := 0;
		n := unsafe.Sizeof(v);
		chain.Printui(uint64(n));
		chain.Printui(database.IDX64);
	}
}`)

	if t.HasMainFunc {
		return nil
	}

	t.writeCode(`
func main() {
	receiver, firstReceiver, action := chain.GetApplyArgs()
	contract := NewContract(receiver, firstReceiver, action)
	if contract == nil {
		return
	}`)

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
		abiFile = t.DirName + "/generated.abi"
	} else {
		abiFile = t.DirName + "/" + t.contractName + ".abi"
	}

	f, err := os.Create(abiFile)
	if err != nil {
		panic(err)
	}

	abi := ABI{}
	abi.Version = "eosio::abi/1.1"
	abi.Structs = make([]ABIStruct, 0, len(t.Structs)+len(t.Actions))

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

	for _, action := range t.Actions {
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

	abi.Actions = make([]ABIAction, 0, len(t.Actions))
	for _, action := range t.Actions {
		a := ABIAction{}
		a.Name = action.ActionName
		a.Type = action.ActionName
		a.RicardianContract = ""
		abi.Actions = append(abi.Actions, a)
	}

	for _, table := range t.Structs {
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

	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if filepath.Ext(path) != ".go" {
			return nil
		}

		if info.Name() == "generated.go" {
			return nil
		}
		goFiles = append(goFiles, path)
		println(path, info.Name())
		return nil
	})
	return goFiles
}

func (t *CodeGenerator) Finish() {
	t.codeFile.Close()
}

func (t *CodeGenerator) Analyse() {
	structMap := make(map[string]*StructInfo)
	for i := range t.Structs {
		s := &t.Structs[i]
		structMap[s.StructName] = s
	}

	t.abiStructsMap = make(map[string]*StructInfo)
	for _, action := range t.Actions {
		for _, member := range action.Members {
			item, ok := structMap[member.Type]
			if ok {
				t.abiStructsMap[member.Type] = item
			}
		}
	}

	for _, item := range t.Structs {
		if item.TableName == "" {
			continue
		}

		for _, member := range item.Members {
			item, ok := structMap[member.Type]
			if ok {
				t.abiStructsMap[member.Type] = item
			}
		}
	}

	for _, item := range t.Structs {
		if item.TableName == "" {
			continue
		}

		item2, ok := structMap[item.StructName]
		if ok {
			t.abiStructsMap[item.StructName] = item2
		}
	}
}

func GenerateCode(inFile string) error {
	gen := NewCodeGenerator()
	gen.fset = token.NewFileSet()

	if filepath.Ext(inFile) == ".go" {
		ext := filepath.Ext(inFile)
		if ext == ".go" {
		}
		gen.DirName = filepath.Dir(inFile)
		if err := gen.ParseGoFile(inFile); err != nil {
			return err
		}
	} else {
		gen.DirName = inFile
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
	if err := gen.GenCode(); err != nil {
		return err
	}
	if err := gen.GenAbi(); err != nil {
		return err
	}
	gen.Finish()
	return nil
}