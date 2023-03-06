package main

import (
	"fmt"

	"github.com/DeniesKresna/gohelper/utlog"
	"github.com/DeniesKresna/myqgen/qgen"
)

func main() {
	listTableColumn := map[string]map[string]string{
		"user": {
			"userID":   "users.id",
			"userName": "users.name",
		},
		"expert": {
			"expertID":     "experts.id",
			"expertUserID": "experts.user_id",
			"expertData":   "experts.data",
		},
	}

	listTable := map[string]string{
		"user":   "users",
		"expert": "experts",
	}

	qGenObj := &qgen.Obj{
		ListTableColumn: listTableColumn,
		ListTable:       listTable,
	}

	query := `SELECT
				<view::user />
				<view::{
					userIdentity > identity: :user.userName;
					userFirstName > firstname: "testdoang";
				} />
				FROM
				<tb:user />
				<join:expert{
					cond: "__::@.userID__ = __::user.userID__ ";
					value: LEFT JOIN;
				} />
				WHERE
				<cond:id[user.userID] /> AND
				<cond:userName[user.userName] />
	`

	args := qgen.Args{
		Fields: []string{
			"userID",
			"userName",
			"userIdentity",
			"userFirstName",
		},
		Conditions: map[string]interface{}{
			"id":            5,
			"userName:LIKE": "%as%",
		},
	}

	res, err := qGenObj.Generate(query, args)
	if err != nil {
		utlog.Error(err)
		return
	}

	fmt.Println(res)
	return
}
