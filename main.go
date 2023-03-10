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
					userFullName > firstname: "users.first_name + user.last_name";
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
				__!sort__
				__!limit__
				__!offset__
	`

	args := qgen.Args{
		Fields: []string{
			"userFullName",
			"user*",
		},
		Conditions: map[string]interface{}{
			"id":             74,
			"firstName:LIKE": "%ar%",
		},
		Sorting: []string{"-userID"},
		Limit:   3,
	}

	res := qGenObj.Build(query, args)

	fmt.Println(res)
	fmt.Printf("\n")

	query2 := `UPDATE
					<tb:user />
				SET
					<set::user />
				WHERE
				<cond:id[user.userID] /> AND
				<cond:firstName[user.userFirstName] />
	`

	args2 := qgen.Args{
		UpdateFields: map[string]interface{}{
			"userFirstName": "administrator",
			"userLastName":  "hebat",
		},
		Conditions: map[string]interface{}{
			"id":             74,
			"firstName:LIKE": "%ar%",
		},
	}

	res2 := qGenObj.Build(query2, args2)

	fmt.Println(res2)

	return
}
