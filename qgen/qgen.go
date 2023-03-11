package qgen

import (
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"errors"

	"github.com/DeniesKresna/gohelper/utinterface"
	"github.com/DeniesKresna/gohelper/utlog"
	"github.com/DeniesKresna/gohelper/utslice"
)

const (

	// tags
	SET_PREFIX   = "<set::"
	VIEW_PREFIX  = "<view::"
	COND_PREFIX  = "<cond::"
	TABLE_PREFIX = "<tb:"
	SORT_PREFIX  = "<sort::"
	JOIN_PREFIX  = "<join:"
	TAG_SUFFIX   = "/>"
	SUB_PREFIX   = "{"
	SUB_SUFFIX   = "}"
	COL_SEP      = ";"

	// states
	QY   = "query"
	VW   = "view"
	CD   = "cond"
	JN   = "join"
	TB   = "table"
	VWBC = "view bracket"
)

var tagSuffixLength = len(TAG_SUFFIX)
var subSuffixLength = len(SUB_SUFFIX)

type Args struct {
	Offset       int64
	Limit        int
	Sorting      []string
	Conditions   map[string]interface{}
	Fields       []string
	Groups       []string
	Distinct     bool
	UpdateFields map[string]interface{}
}

type Obj struct {
	ListTableColumn map[string]map[string]string
	ListTable       map[string]string
	IsLogged        bool
}

func InitObject(isLogged bool, tables ...interface{}) (obj *Obj, err error) {
	obj = &Obj{
		ListTableColumn: make(map[string]map[string]string),
		ListTable:       make(map[string]string),
	}
	var (
		listTable       = make(map[string]string)
		listTableColumn = make(map[string]map[string]string)
	)

	for _, tbSt := range tables {
		var tbVal = reflect.ValueOf(tbSt)
		if !utinterface.IsStruct(tbVal) {
			err = errors.New("Argument should be struct")
			return
		}

		var (
			tbName, tbAlias string
		)

		tbNameRes := tbVal.MethodByName("GetTableNameAndAlias").Call([]reflect.Value{})
		tbName = tbNameRes[0].Interface().(string)
		tbAlias = tbNameRes[1].Interface().(string)
		if len(tbNameRes) < 2 && tbName == "" && tbAlias == "" {
			err = errors.New("Error when get table name and alias")
			return
		}

		listTable[tbAlias] = tbName
		listTableColumn[tbAlias] = make(map[string]string)

		var reflectType = tbVal.Type()

		for i := 0; i < reflectType.NumField(); i++ {
			fieldTags := reflectType.Field(i).Tag

			sqlqTagStr := fieldTags.Get("sqlq")
			if len(sqlqTagStr) <= 1 || sqlqTagStr == "" {
				continue
			}
			sqlqTags := strings.Split(sqlqTagStr, ",")

			sqlqTag := sqlqTags[0]

			dbTagStr := fieldTags.Get("jsondb")
			dbTag := ""
			if dbTagStr != "" {
				dbTags := strings.Split(dbTagStr, ",")
				if len(dbTags) > 0 {
					dbTag = dbTags[0]
				}
				childList := strings.Split(dbTag, ".")
				if len(childList) > 1 {
					for idx, v := range childList {
						if idx == 0 {
							dbTag = fmt.Sprintf("`%s`->>'$", v)
							continue
						}
						dbTag += fmt.Sprintf(".%s", v)
					}
					dbTag += "'"
				}
			} else {
				dbTagStr = fieldTags.Get("db")
				dbTags := strings.Split(dbTagStr, ",")
				if len(dbTags) > 0 {
					dbTag = dbTags[0]
				}
			}

			if dbTag != "" {
				listTableColumn[tbAlias][sqlqTag] = fmt.Sprintf("%s.%s", tbName, dbTag)
			}
		}
	}
	obj.ListTableColumn = listTableColumn
	obj.ListTable = listTable
	obj.IsLogged = isLogged

	return
}

func (q *Obj) HandleGenerateSetTag(query string, args Args) (res string, err error) {
	lastIndex := len(query)
	viewVar := strings.Split(query, "::")

	if len(viewVar) < 2 {
		err = errors.New("Set tag should has table or column alias")
		return
	}

	var jsonFieldMap = make(map[string]string)
	tb := query[len(SET_PREFIX) : lastIndex-len(TAG_SUFFIX)]
	tb = strings.TrimSpace(tb)
	listColumn, ok := q.ListTableColumn[tb]
	if ok {
		var hasSet = make(map[string]bool)
		for key, uf := range args.UpdateFields {
			if val, ok2 := listColumn[key]; ok2 {
				if _, ok3 := hasSet[val]; !ok3 {
					if strings.Contains(val, "->") {
						if _, ok4 := jsonFieldMap[key]; !ok4 {
							jsonFieldMap[key] = val
							continue
						}
					}
					res += fmt.Sprintf("%s = %v, ", val, ConvertToEscapeString(uf, ""))
					hasSet[val] = true
				}
			}
		}
	}

	if len(jsonFieldMap) > 0 {
		var jsonSet = make(map[string]string)
		for key, realField := range jsonFieldMap {
			jsonStructure := strings.Split(realField, "->>")
			rootCol := strings.ReplaceAll(jsonStructure[0], "`", "")

			var tmpRes string
			if _, ok1 := jsonSet[rootCol]; !ok1 {
				tmpRes = fmt.Sprintf("%s = JSON_SET( %s", rootCol, rootCol)
			}

			tmpRes += fmt.Sprintf(", %s, %s", jsonStructure[1], ConvertToEscapeString(args.UpdateFields[key], ""))
			jsonSet[rootCol] = jsonSet[rootCol] + tmpRes
		}

		for _, v := range jsonSet {
			if v == "" {
				continue
			}
			res += fmt.Sprintf("%s),", v)
		}
	}

	return
}

func (q *Obj) HandleGenerateViewTag(query string, args Args) (res string, additionalField map[string]string, err error) {
	additionalField = make(map[string]string)

	lastIndex := len(query)
	viewVar := strings.Split(query, "::")

	if len(viewVar) < 2 {
		err = errors.New("View tag should has table or column alias")
		return
	}

	tb := query[len(VIEW_PREFIX) : lastIndex-len(TAG_SUFFIX)]
	tb = strings.TrimSpace(tb)
	listColumn, ok := q.ListTableColumn[tb]
	for k, v := range q.ListTableColumn[tb] {
		additionalField[k] = v
	}
	if ok {
		for _, f := range args.Fields {
			if val, ok2 := listColumn[f]; ok2 {
				res += fmt.Sprintf("%s, ", val)
			}
		}
	}
	return
}

func (q *Obj) HandleGenerateViewCurly(query string, args Args) (res string, additionalField map[string]string, err error) {
	additionalField = make(map[string]string)

	viewVar := strings.Split(query, "::")

	if len(viewVar) < 2 {
		err = errors.New("View tag should has table or column alias")
		return
	}

	curlInitIdx := strings.Index(query, SUB_PREFIX)
	curlEndIdx := strings.Index(query, SUB_SUFFIX)

	colList := query[curlInitIdx+1 : curlEndIdx]
	colList = strings.TrimSpace(colList)

	cols := strings.Split(colList, COL_SEP)

	for _, col := range cols {
		col = strings.TrimSpace(col)
		if col == "" {
			continue
		}

		colDes := strings.Split(col, ":")
		lenColDes := len(colDes)
		var (
			colKey, colVal string
		)
		if lenColDes < 2 {
			continue
		} else if lenColDes == 2 {
			colKey, colVal = strings.TrimSpace(colDes[0]), strings.TrimSpace(colDes[1])
		} else if lenColDes == 3 {
			colKey, colVal = strings.TrimSpace(colDes[0]), strings.TrimSpace(colDes[2])
		} else {
			continue
		}

		colKeyDes := strings.Split(colKey, ">")
		if len(colDes) < 2 {
			err = errors.New("Additional column should has alias")
			continue
		}

		colField, colAlias := strings.TrimSpace(colKeyDes[0]), strings.TrimSpace(colKeyDes[1])
		if !utslice.IsExist(args.Fields, colField) {
			continue
		}

		if strings.HasPrefix(colVal, `"`) && strings.HasSuffix(colVal, `"`) {
			res += fmt.Sprintf("%s AS %s, ", strings.TrimSpace(colVal[1:len(colVal)-1]), colAlias)
			additionalField[colField] = colAlias
		} else {
			colValDes := strings.Split(colVal, ".")
			if len(colValDes) != 2 {
				err = errors.New(fmt.Sprintf("Column not found: %s", colVal))
				continue
			}
			tb := colValDes[0]

			var (
				tbCol map[string]string
				ok2   bool
			)

			if tbCol, ok2 = q.ListTableColumn[tb]; !ok2 {
				err = errors.New(fmt.Sprintf("Table not found: %s", tb))
				continue
			}

			if val, ok3 := tbCol[colValDes[1]]; ok3 {
				res += fmt.Sprintf("%s AS %s, ", val, colAlias)
				additionalField[colField] = colAlias
			}
		}
	}
	return
}

func (q *Obj) HandleGenerateTable(query string) (res string, err error) {
	lastIndex := len(query)
	viewVar := strings.Split(query, ":")

	if len(viewVar) < 2 {
		err = errors.New("Table tag should has this format <tb:tablename />")
		return
	}

	tb := query[len(TABLE_PREFIX) : lastIndex-len(TAG_SUFFIX)]
	tb = strings.TrimSpace(tb)
	res, ok := q.ListTable[tb]
	if !ok {
		err = errors.New(fmt.Sprintf("Table %s not found", tb))
		return
	}

	return
}

func (q *Obj) HandleGenerateJoin(query string) (res string, err error) {
	joinVar := strings.Split(query, ":")

	const (
		COND_KEY  = "cond:"
		VALUE_KEY = "value:"
	)

	if len(joinVar) < 2 {
		err = errors.New("Join tag should has this format <join:tablename{freeQuery} />")
		return
	}

	joinInitCtn := strings.Index(query, SUB_PREFIX)
	joinEndCtn := strings.Index(query, SUB_SUFFIX)

	tb := query[len(JOIN_PREFIX):joinInitCtn]
	tb = strings.TrimSpace(tb)

	realTb, ok := q.ListTable[tb]
	if !ok {
		err = errors.New(fmt.Sprintf("Table %s not found", tb))
		return
	}

	joinCtn := query[joinInitCtn+1 : joinEndCtn]
	joinCtnDes := strings.Split(joinCtn, COL_SEP)

	var joinCond, joinValue string

	for _, ctn := range joinCtnDes {
		ctn := strings.TrimSpace(ctn)
		if len(ctn) <= 3 {
			continue
		}
		ctn = strings.TrimSpace(ctn)
		if strings.HasPrefix(ctn, COND_KEY) {
			ctn = ctn[len(COND_KEY):]
			ctn = strings.ReplaceAll(ctn, "@", tb)
			sent := regexp.MustCompile(DOB_QUOTE)
			matches := sent.FindAllStringSubmatchIndex(ctn, -1)
			for _, match := range matches {
				quoteStr := ctn[match[0]:match[1]]
				joinCond = strings.TrimSpace(quoteStr[1 : len(quoteStr)-1])
				break
			}
		} else if strings.HasPrefix(ctn, VALUE_KEY) {
			ctnVarDes := strings.Split(ctn, ":")
			if len(ctnVarDes) < 2 {
				err = errors.New("join value should has value")
				return
			}
			ctnVarVal := strings.ToUpper(strings.TrimSpace(ctnVarDes[1]))
			if ctnVarVal == "INNER JOIN" || ctnVarVal == "LEFT JOIN" || ctnVarVal == "RIGHT JOIN" || ctnVarVal == "JOIN" {
				joinValue = ctnVarVal
			}
		}
	}
	if joinValue == "" {
		joinValue = "INNER JOIN"
	}

	res = fmt.Sprint(" ", joinValue, " ", realTb, " ON ", joinCond, " ")
	return
}

func (q *Obj) HandleGenerateCondPlain(query string, condFieldList map[string]string) (res string, err error) {
	condVar := strings.Split(query, ":")

	if len(condVar) < 2 {
		err = errors.New("Cond tag should has this format <cond:field[realField] />")
		return
	}

	condVar1 := condVar[1]
	fields := strings.Split(condVar1, "[")

	if len(fields) < 2 {
		err = errors.New("Cond tag should has this format <cond:field[realField] />")
		return
	}

	fieldFilter, fieldColumn := fields[0], fields[1]
	condition, ok := condFieldList[fieldFilter]
	if !ok {
		res += "TRUE "
		return
	}

	endOfBoxBracket := strings.Index(fieldColumn, "]")
	if endOfBoxBracket < 0 {
		err = errors.New("Cond tag should has ] syntax />")
		return
	}

	tableField := fieldColumn[:endOfBoxBracket]
	tableFieldDes := strings.Split(tableField, ".")

	if len(fields) != 2 {
		err = errors.New("Field should has this format table.field")
		return
	}

	tb := tableFieldDes[0]
	tbCol, ok2 := q.ListTableColumn[tb]
	if !ok2 {
		err = errors.New(fmt.Sprintf("Table not found: %s", tb))
		return
	}

	realField, ok3 := tbCol[tableFieldDes[1]]
	if ok3 {
		res += fmt.Sprintf("%s %s ", realField, condition)
	}

	return
}

func (q *Obj) ResolveFinishing(query string, args Args, additionalField map[string]string) (res string, err error) {
	res = query
	sent := regexp.MustCompile(TABLE_VAR)
	matches := sent.FindAllStringSubmatchIndex(query, -1)

	var varsMap = make(map[string]string)
	for _, v := range matches {
		theVar := query[v[0]:v[1]]
		theVarDes := strings.Split(theVar, ".")
		tbAlias := theVarDes[0][4:]
		_, ok := q.ListTable[tbAlias]
		if !ok {
			err = errors.New(fmt.Sprintf("Table %s not found", tbAlias))
			return
		}

		colAlias := theVarDes[1][:len(theVarDes[1])-2]
		col, ok1 := q.ListTableColumn[tbAlias][colAlias]
		if !ok1 {
			err = errors.New(fmt.Sprintf("Column %s not found", colAlias))
			return
		}

		if _, ok2 := varsMap[theVar]; !ok2 {
			varsMap[theVar] = fmt.Sprintf("%s", col)
			res = strings.ReplaceAll(res, theVar, varsMap[theVar])
		}
	}

	res = strings.ReplaceAll(res, ", FROM ", " FROM ")
	res = strings.ReplaceAll(res, ", WHERE ", " WHERE ")

	if args.Distinct {
		res = strings.ReplaceAll(res, "__!distinct__", "DISTINCT")
	} else {
		res = strings.ReplaceAll(res, "__!distinct__", "")
	}

	if args.Limit >= 0 {
		res = strings.ReplaceAll(res, "__!limit__", fmt.Sprintf("LIMIT %d", args.Limit))
	} else {
		res = strings.ReplaceAll(res, "__!limit__", "")
	}

	if args.Offset > 0 && args.Limit != 0 {
		res = strings.ReplaceAll(res, "__!offset__", fmt.Sprintf("OFFSET %d", args.Offset))
	} else {
		res = strings.ReplaceAll(res, "__!offset__", "")
	}

	if len(args.Sorting) > 0 {
		sortRes := "ORDER BY"
		for idx, v := range args.Sorting {
			isDesc := false
			if strings.HasPrefix(v, "-") {
				isDesc = true
			}
			v = strings.TrimPrefix(v, "-")
			realField, ok := additionalField[v]
			if ok {
				sortRes += fmt.Sprintf(" %s ", realField)
				if isDesc {
					sortRes += "DESC"
				} else {
					sortRes += "ASC"
				}
				if idx < len(args.Sorting)-1 {
					sortRes += ","
				}
			}
		}
		res = strings.ReplaceAll(res, "__!sort__", sortRes)
	} else {
		res = strings.ReplaceAll(res, "__!sort__", "")
	}

	if q.IsLogged {
		fmt.Printf("SQL DEBUG: \n%s\n", res)
	}

	return
}

func (q *Obj) Build(query string, args Args) (res string) {
	regexPatterns := map[string]string{
		"viewDefault": VIEW_DEFAULT,
		"viewCurly":   VIEW_CURLY,
		"table":       TABLE,
		"join":        JOIN,
		"condPlain":   COND_PLAIN,
		"condModif":   COND_MODIF,
		"setDefault":  SET_DEFAULT,
	}

	if args.Limit <= 0 {
		args.Limit = -1
	}

	var (
		matchMap = make(map[int]map[string]interface{})
		err      error
	)

	for tagType, pattern := range regexPatterns {
		sent := regexp.MustCompile(pattern)
		matches := sent.FindAllStringSubmatchIndex(query, -1)
		for _, v := range matches {
			matchMap[v[0]] = map[string]interface{}{
				"type":  tagType,
				"range": v,
			}
		}
	}

	var (
		i          int = -1
		recordFlag     = false
		earlyIdx       = 0
	)
	for i < len(query)-1 {
		i++
		if val, ok := matchMap[i]; !ok {
			if !recordFlag {
				earlyIdx = i
				recordFlag = true
			}
		} else {
			if recordFlag {
				recordFlag = false
				matchMap[earlyIdx] = map[string]interface{}{
					"type":  "free",
					"range": query[earlyIdx:i],
				}
			}
			i = val["range"].([]int)[1]
		}
	}
	if recordFlag {
		matchMap[earlyIdx] = map[string]interface{}{
			"type":  "free",
			"range": query[earlyIdx:],
		}
	}

	for k, v := range args.Fields {
		if strings.HasSuffix(v, "*") {
			starIndex := strings.Index(v, "*")
			tbAlias := v[:starIndex]
			tbAlias = strings.TrimSpace(tbAlias)

			ltc := q.ListTableColumn[tbAlias]

			ix := 0
			for k2, v2 := range ltc {
				_ = v2
				if ix == 0 {
					args.Fields[k] = k2
				} else {
					args.Fields = append(args.Fields, k2)
				}
				ix++
			}
		}
	}

	var condFieldList = make(map[string]string)
	for k, v := range args.Conditions {
		condVar := strings.Split(k, ":")
		condVarLen := len(condVar)
		if condVarLen == 1 {
			if _, ok := condFieldList[condVar[0]]; !ok {
				condFieldList[condVar[0]] = "= " + ConvertToEscapeString(v, "")
			}
		} else if condVarLen == 2 {
			if _, ok := condFieldList[condVar[0]]; !ok {
				condFieldList[condVar[0]] = strings.ToUpper(condVar[1]) + " " + ConvertToEscapeString(v, "")
			}
		}
	}

	listKey := []int{}
	for k, v := range matchMap {
		_ = v
		listKey = append(listKey, k)
	}

	sort.Ints(listKey)

	var additionalField = make(map[string]string)

	for _, v := range listKey {
		mat := matchMap[v]
		switch mat["type"].(string) {
		case "viewDefault":
			rng := mat["range"].([]int)
			var (
				resf      string
				addFields map[string]string
			)

			resf, addFields, err = q.HandleGenerateViewTag(query[rng[0]:rng[1]], args)
			if err != nil {
				utlog.Error(err)
				return
			}

			for k2, v2 := range addFields {
				additionalField[k2] = v2
			}

			res += resf
		case "viewCurly":
			rng := mat["range"].([]int)
			var (
				resf      string
				addFields map[string]string
			)

			resf, addFields, err = q.HandleGenerateViewCurly(query[rng[0]:rng[1]], args)
			if err != nil {
				utlog.Error(err)
				return
			}

			for k2, v2 := range addFields {
				additionalField[k2] = v2
			}

			res += resf
		case "table":
			rng := mat["range"].([]int)
			var resf string
			resf, err = q.HandleGenerateTable(query[rng[0]:rng[1]])
			if err != nil {
				utlog.Error(err)
				return
			}
			res += resf
		case "join":
			rng := mat["range"].([]int)
			var resf string
			resf, err = q.HandleGenerateJoin(query[rng[0]:rng[1]])
			if err != nil {
				utlog.Error(err)
				return
			}
			res += resf
		case "condPlain":
			rng := mat["range"].([]int)
			var resf string
			resf, err = q.HandleGenerateCondPlain(query[rng[0]:rng[1]], condFieldList)
			if err != nil {
				utlog.Error(err)
				return
			}
			res += resf
		case "setDefault":
			rng := mat["range"].([]int)
			var resf string

			resf, err = q.HandleGenerateSetTag(query[rng[0]:rng[1]], args)
			if err != nil {
				utlog.Error(err)
				return
			}
			res += resf
		default:
			res += mat["range"].(string)
		}
	}

	res = strings.Join(strings.Fields(res), " ")
	res, err = q.ResolveFinishing(res, args, additionalField)
	if err != nil {
		utlog.Error(err)
		return
	}

	return
}
