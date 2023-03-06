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
	Offset     int64
	Limit      int
	Sorting    []string
	Conditions map[string]interface{}
	Fields     []string
	Groups     []string
}

type Obj struct {
	ListTableColumn map[string]map[string]string
	ListTable       map[string]string
}

func (q *Obj) RegisStruct(tables []interface{}) (err error) {
	for _, tbSt := range tables {
		var tbVal = reflect.ValueOf(tbSt)
		if !utinterface.IsStruct(tbVal) {
			err = errors.New("Argument should be struct")
			return
		}

		var (
			tbName, tbAlias string
		)

		tbNameRes := tbVal.MethodByName("GetTableName").Call([]reflect.Value{})
		tbName = tbNameRes[0].Interface().(string)
		if len(tbNameRes) < 1 && tbNameRes[0].Interface().(string) == "" {
			err = errors.New("Error when get table name")
			return
		}

		tbAliasRes := tbVal.MethodByName("GetTableAlias").Call([]reflect.Value{})
		tbAlias = tbAliasRes[0].Interface().(string)
		if len(tbAliasRes) < 1 && tbAliasRes[0].Interface().(string) == "" {
			err = errors.New("Error when get table alias")
			return
		}

		q.ListTable[tbAlias] = tbName

		var reflectType = tbVal.Type()

		for i := 0; i < reflectType.NumField(); i++ {
			fieldName := reflectType.Field(i).Name
			fieldTags := reflectType.Field(i).Tag

			sqlqTagStr := fieldTags.Get("sqlq")
			sqlqTags := strings.Split(sqlqTagStr, ",")
			if len(sqlqTagStr) <= 1 {
				err = errors.New(fmt.Sprintf("Should has sqlq tag in field: %s", fieldName))
				return
			}
			sqlqTag := sqlqTags[0]

			dbTagStr := fieldTags.Get("db")
			dbTags := strings.Split(dbTagStr, ",")
			if len(dbTagStr) <= 1 {
				err = errors.New(fmt.Sprintf("Should has db tag in field: %s", fieldName))
				return
			}
			dbTag := dbTags[0]

			q.ListTableColumn[tbAlias][sqlqTag] = fmt.Sprintf("%s.%s", tbName, dbTag)
		}
	}
	return
}

func (q *Obj) HandleGenerateViewTag(query string, args Args, isEndView bool) (res string, err error) {
	lastIndex := len(query)
	viewVar := strings.Split(query, "::")

	if len(viewVar) < 2 {
		err = errors.New("View tag should has table or column alias")
		return
	}

	tb := query[len(VIEW_PREFIX) : lastIndex-len(TAG_SUFFIX)]
	tb = strings.TrimSpace(tb)
	listColumn, ok := q.ListTableColumn[tb]
	if ok {
		for _, f := range args.Fields {
			if val, ok2 := listColumn[f]; ok2 {
				if isEndView {
					res += fmt.Sprintf("%s ", val)
					return
				}
				res += fmt.Sprintf("%s, ", val)
			}
		}
	}
	return
}

func (q *Obj) HandleGenerateViewCurly(query string, args Args, isEndView bool) (res string, err error) {
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

	for idx, col := range cols {
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
			if isEndView {
				if idx+2 <= len(cols) {
					if cols[idx+1] == "" {
						res += fmt.Sprintf("%s AS %s ", strings.TrimSpace(colVal[1:len(colVal)-1]), colAlias)
						continue
					}
				}
			}
			res += fmt.Sprintf("%s AS %s, ", strings.TrimSpace(colVal[1:len(colVal)-1]), colAlias)
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
				if isEndView {
					if idx+2 <= len(cols) {
						if cols[idx+1] == "" {
							res += fmt.Sprintf("%s AS %s ", val, colAlias)
							continue
						}
					}
				}
				res += fmt.Sprintf("%s AS %s, ", val, colAlias)
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

func (q *Obj) Generate(query string, args Args) (res string, err error) {
	regexPatterns := map[string]string{
		"viewDefault": VIEW_DEFAULT,
		"viewCurly":   VIEW_CURLY,
		"table":       TABLE,
		"join":        JOIN,
		"condPlain":   COND_PLAIN,
		"condModif":   COND_MODIF,
	}

	var matchMap = make(map[int]map[string]interface{})

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

	var condFieldList = make(map[string]string)
	for k, v := range args.Conditions {
		condVar := strings.Split(k, ":")
		condVarLen := len(condVar)
		if condVarLen == 1 {
			if _, ok := condFieldList[condVar[0]]; !ok {
				condFieldList[condVar[0]] = "= " + ConvertToString(v, "")
			}
		} else if condVarLen == 2 {
			if _, ok := condFieldList[condVar[0]]; !ok {
				condFieldList[condVar[0]] = strings.ToUpper(condVar[1]) + " " + ConvertToString(v, "")
			}
		}
	}

	listKey := []int{}
	for k, v := range matchMap {
		_ = v
		listKey = append(listKey, k)
	}

	sort.Ints(listKey)

	for idx, v := range listKey {
		mat := matchMap[v]
		switch mat["type"].(string) {
		case "viewDefault":
			rng := mat["range"].([]int)
			var (
				resf      string
				isEndView bool
			)

			if idx+2 <= len(listKey) {
				nexMat := matchMap[listKey[idx+1]]
				tagGroup := nexMat["type"].(string)
				if tagGroup == "free" {
					vl := strings.ToLower(strings.TrimSpace(nexMat["range"].(string)))

					if strings.HasPrefix(vl, "from") {
						isEndView = true
					}
				}
			}

			resf, err = q.HandleGenerateViewTag(query[rng[0]:rng[1]], args, isEndView)
			if err != nil {
				utlog.Error(err)
				return
			}
			res += resf
		case "viewCurly":
			rng := mat["range"].([]int)
			var (
				resf      string
				isEndView bool
			)

			if idx+2 <= len(listKey) {
				nexMat := matchMap[listKey[idx+1]]
				tagGroup := nexMat["type"].(string)
				if tagGroup == "free" {
					vl := strings.ToLower(strings.TrimSpace(nexMat["range"].(string)))

					if strings.HasPrefix(vl, "from") {
						isEndView = true
					}
				}
			}

			resf, err = q.HandleGenerateViewCurly(query[rng[0]:rng[1]], args, isEndView)
			if err != nil {
				utlog.Error(err)
				return
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
		default:
			res += mat["range"].(string)
		}
	}

	res = strings.Join(strings.Fields(res), " ")

	return
}
