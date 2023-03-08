package main

import (
	"fmt"

	"github.com/DeniesKresna/myqgen/qgen"
)

func main() {
	listTableColumn := map[string]map[string]string{
		"user": {
			"userID":        "users.id",
			"userFirstName": "users.first_name",
			"userLastName":  "users.last_name",
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
				__!distinct__
				<view::user />
				<view::{
					userIdentity > identity: :user.userName;
					userFirstName > firstname: "users.first_name";
				} />
				FROM
				<tb:user />
				<join:expert{
					cond: "__::@.expertUserID__ = __::user.userID__ ";
					value: LEFT JOIN;
				} />
				WHERE
				<cond:id[user.userID] /> AND
				<cond:firstName[user.userFirstName] />
				__!limit__
				__!offset__
	`

	args := qgen.Args{
		Fields: []string{
			"userIdentity",
			"user*",
		},
		Conditions: map[string]interface{}{
			"id":             74,
			"firstName:LIKE": "%ar%",
		},
		Limit: 3,
	}

	res := qGenObj.Build(query, args)

	fmt.Println(res)
	return
}
