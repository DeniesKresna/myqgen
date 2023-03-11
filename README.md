# Myqgen

MySQL Query Generator

## Description

This is simple mysql query generator for doing some simple query in my owned company for easy query and readable code.
Please use it just for simple query. will not working on complicated query. i suggest use native query instead.

Inspired from sqlq package authored by my copartner [GearIntellix](https://github.com/gearintellix) in my full time job at one of biggest p2p lending company in Indonesia.
Its not fully similar, i used with my preference style.
This is genuine code from me, can be enhanced a lot because i made it quick for chasing project deadlines. Thanks for anyone who want give me advices.

## Getting Started

### Dependencies

I used this on go v18 projects. But you can use lower.

### How to use

* Just get the package by ```go get github.com/DeniesKresna/myqgen``` in terminal
* Prepare some struct as referrence of the table. Set the sqlq tag, it is the important part. Dont forget to set GetTableNameAndAlias method it will be used when init the object.
```
type User struct {
	ID        int64      `json:"id" db:"id" sqlq:"userID"`
	CreatedBy string     `json:"created_by" db:"created_by" sqlq:"userCreatedBy"`
	CreatedAt time.Time  `json:"created_at" db:"created_at" sqlq:"userCreatedAt"`
	UpdatedBy string     `json:"updated_by" db:"updated_by" sqlq:"userUpdatedBy"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at" sqlq:"userUpdatedAt"`
	DeletedBy *string    `json:"deleted_by" db:"deleted_by" sqlq:"userDeletedBy"`
	DeletedAt *time.Time `json:"deleted_at" db:"deleted_at" sqlq:"userDeletedAt"`
	FirstName string     `json:"first_name" db:"first_name" sqlq:"userFirstName"`
	LastName  string     `json:"last_name" db:"last_name" sqlq:"userLastName"`
	Email     string     `json:"email" db:"email" sqlq:"userEmail"`
	Phone     string     `json:"phone" db:"phone" sqlq:"userPhone"`
	ImageUrl  *string    `json:"image_url" db:"image_url" sqlq:"userImageURL"`
	Password  string     `json:"-" db:"password" sqlq:"userPassword"`
}

func (u User) GetTableNameAndAlias() (string, string) {
	return "users", "user"
}
```
* Register the table in qgen (query generator) Obj
```
import ""github.com/DeniesKresna/myqgen/qgen"

q, err := qgen.InitObject(isLogged, types.User{}, types.Role{})
if err != nil {
	return
}
```
set isLogged value to true (boolean) if you want to check generated query in console.
* Set the query pattern
```
package queries

const GetUser = `
			SELECT
				__!distinct__
				<view::user />
				<view::{
					roleName > role_name: :role.roleName;
				} />
			FROM
				<tb:user />
				<join:role{
					cond: "__::@.roleID__ = __::user.userRoleID__ ";
					value: INNER JOIN;
				} />
			WHERE
				<cond:id[user.userID] /> AND
				<cond:email[user.userEmail] /> AND
				<cond:active[user.userActive] /> AND
				__::user.userDeletedAt__ IS NULL
			__!limit__
			__!offset__
		`
```
for helping you work in vs code you can install [sqlq](https://marketplace.visualstudio.com/items?itemName=GearIntellix.vscode-sqml) package by GearIntellix
* Use the q object in wherever part you want.
```
query := r.q.Build(queries.GetUser, qgen.Args{
		Fields: []string{
			"userID",
			"userCreatedAt",
			"userUpdatedAt",
			"userDeletedAt",
			"userFirstName",
			"userLastName",
			"userEmail",
			"userPhone",
			"userImageURL",
			"userRoleID",
			"roleName",
		},
		Conditions: map[string]interface{}{
			"id": 1,
		},
})
```

it will generate code look like 
```
SELECT  users.id, users.created_at, users.updated_at, users.deleted_at, users.first_name, users.last_name, users.email, users.phone, users.image_url, users.role_id, roles.name AS role_name FROM users INNER JOIN roles ON roles.id = users.role_id WHERE users.id = 1 AND TRUE AND TRUE AND users.deleted_at IS NULL 
```
You can use whatever mysql executer package you want.

### Simple Documentation

* __!distinct__, __!limit__, __!offset__ you can use this in the query, i will be ignored if you didnt set any data to the related ```qgen.Args key```
* <view::"table Alias" /> table alias is the alias will be used in the logical code. it refer to alias you set in ```GetTableNameAndAlias``` method
  table columns will be selected in query as long you put in args.Fields based on sqlq tag in the struct. (see code above)
* in this pattern 
```<view::{ "fieldChosen" > "alias in query": :"sqlq column"; } />```
    you can include this column if you set the "field Chosen" in args.Field as well.
* for <cond:id[user.userID] /> same with fields, table columns will be used to filter in query as long you put in args.Conditions based on sqlq tag in the struct. (see code above)
* for another operator condition you can use pattern like this example. ```"id:>": 1,``` or ```"id:IN": []int{1,2,3}```

## Authors

DeniesKresna

## License

-

## Help
For question or advice you can email me at denieskresna@gmail.com
