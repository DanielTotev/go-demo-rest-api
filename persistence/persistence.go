package persistence

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	_ "modernc.org/sqlite"
)

var db *sql.DB

func GetDb() *sql.DB {
	return db
}

func Init() {
	var err error
	db, err = sql.Open("sqlite", "local.db")
	if err != nil {
		fmt.Println(err)
		fmt.Println("===============")
		panic("No database connection")
	}
	db.SetMaxOpenConns(10)
}

func CreateTable(structType reflect.Type) {
	var field reflect.StructField
	var columnName string
	var fieldType string
	var primaryKey bool
	var sql strings.Builder
	sql.WriteString("CREATE TABLE IF NOT EXISTS " + strings.ToLower(structType.Name()) + " (\n")
	for i, n := 0, structType.NumField(); i < n; i++ {
		field = structType.Field(i)
		columnName = field.Tag.Get("name")
		if columnName == "" {
			fmt.Println("Skipping field " + field.Name + " it does not have the attribute column name and for this reason is treated as non persistent")
			continue
		}
		primaryKey = field.Tag.Get("pk") == "true"
		fieldType = getSQLType(field.Type.Name())
		sql.WriteString(fmt.Sprintf("  %v %v", columnName, fieldType))
		if primaryKey {
			sql.WriteString(" PRIMARY KEY AUTOINCREMENT")
		}
		if i != n-1 {
			sql.WriteString(",\n")
		} else {
			sql.WriteString("\n")
		}
	}
	sql.WriteString(")")
	_, err := db.Exec(sql.String())
	if err != nil {
		fmt.Print(err.Error())
		panic("Could not create table")
	}
}

func Insert[T any](obj *T) error {
	return upsert(obj, false)
}

func Update[T any](obj *T) error {
	return upsert(obj, true)
}

func LoadAll[T any](structType reflect.Type) ([]T, error) {
	tableName := strings.ToLower(structType.Name())
	sql := fmt.Sprintf("SELECT * FROM %v", tableName)
	cursor, err := db.Query(sql)
	if err != nil {
		return nil, err
	}
	defer cursor.Close()
	return parseCursorResults[T](cursor, structType)
}

func LoadByPrimaryKeyValue[T any](structType reflect.Type, pkValue any) (T, error) {
	var result T
	var tableName = strings.ToLower(structType.Name())
	pkColumnName := findPkColumnName(structType)
	stmt, err := db.Prepare(fmt.Sprintf("SELECT * FROM %v WHERE %v = ?", tableName, pkColumnName))
	if err != nil {
		return result, err
	}
	cursor, err := stmt.Query(pkValue)
	if err != nil {
		return result, err
	}
	defer cursor.Close()
	defer stmt.Close()
	res, err := parseCursorResults[T](cursor, structType)
	if err != nil {
		return result, err
	}
	if len(res) > 0 {
		return res[0], nil
	}
	return result, nil
}

func DeleteByPrimaryKeyValue(obj any) error {
	objType := reflect.TypeOf(obj)
	objValues := reflect.ValueOf(obj)
	primaryKeyColumnName := findPkColumnName(objType)
	pkValue := objValues.FieldByName(primaryKeyColumnName).Interface()
	sql := fmt.Sprintf("DELETE FROM %v WHERE %v = ?", objType.Name(), primaryKeyColumnName)
	stmt, err := db.Prepare(sql)
	if err != nil {
		return err
	}
	_, execErr := stmt.Exec(pkValue)
	if execErr != nil {
		return execErr
	}
	return nil
}

func parseCursorResults[T any](cursor *sql.Rows, structType reflect.Type) ([]T, error) {
	var result []T
	for cursor.Next() {
		newInstance := reflect.New(structType).Elem()
		var newInstancePointers []any
		for i, n := 0, newInstance.NumField(); i < n; i++ {
			newInstancePointers = append(newInstancePointers, newInstance.Field(i).Addr().Interface())
		}
		scanErr := cursor.Scan(newInstancePointers...)
		if scanErr != nil {
			return nil, scanErr
		}
		result = append(result, newInstance.Interface().(T))
	}
	return result, nil
}

func findPkColumnName(structType reflect.Type) string {
	var pkColumnName string
	for i, n := 0, structType.NumField(); i < n; i++ {
		if structType.Field(i).Tag.Get("pk") == "true" {
			pkColumnName = structType.Field(i).Name
			break
		}
	}
	return pkColumnName
}

func upsert[T any](obj *T, isUpdate bool) error {
	objValue := reflect.ValueOf(obj)
	if objValue.Kind() != reflect.Ptr || objValue.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("expected a pointer to a struct")
	}

	objElem := objValue.Elem()
	objType := objElem.Type()

	var columns []string
	var values []any
	var primaryKeyColumnName string
	var primaryKeyColumnValue any
	var primaryKeyFieldName string

	tableName := strings.ToLower(objType.Name())

	// Iterate over the struct fields
	for i := 0; i < objType.NumField(); i++ {
		field := objType.Field(i)
		columnName := field.Tag.Get("name")

		if columnName == "" {
			fmt.Println("Skipping field " + field.Name + " it does not have the attribute column name and for this reason is treated as non persistent")
			continue
		}

		primaryKey := field.Tag.Get("pk") == "true"
		fieldValue := objElem.Field(i).Interface()

		if primaryKey {
			primaryKeyColumnName = columnName
			primaryKeyColumnValue = fieldValue
			primaryKeyFieldName = field.Name
		} else {
			columns = append(columns, columnName)
			values = append(values, fieldValue)
		}
	}

	var sql string

	// Build the SQL query
	if !isUpdate {
		sql = fmt.Sprintf("INSERT INTO %v (%v) VALUES (%v)", tableName, strings.Join(columns, ", "), generateParameterPlaceHolders(len(columns)))
	} else {
		var sqlBuilder = buildUpdateSql(tableName, columns, primaryKeyColumnName)
		values = append(values, primaryKeyColumnValue)
		sql = sqlBuilder.String()
	}
	fmt.Println(sql)

	stmt, err := db.Prepare(sql)
	if err != nil {
		return err
	}
	res, stmtErr := stmt.Exec(values...)
	if stmtErr != nil {
		return stmtErr
	}
	defer stmt.Close()

	// Set the primary key field with the last insert ID if it's an insert operation
	if !isUpdate {
		id, err := res.LastInsertId()
		if err != nil {
			return err
		}
		// Set the primary key field with the last inserted ID
		pkField := objElem.FieldByName(primaryKeyFieldName)
		if pkField.CanSet() {
			pkField.Set(reflect.ValueOf(id).Convert(pkField.Type()))
		} else {
			return fmt.Errorf("cannot set primary key field %s", primaryKeyFieldName)
		}
	}

	return nil
}

func buildUpdateSql(tableName string, columns []string, primaryKeyColumnName string) strings.Builder {
	var sqlBuilder strings.Builder
	sqlBuilder.WriteString(fmt.Sprintf("UPDATE %v SET ", tableName))
	for i, column := range columns {
		sqlBuilder.WriteString(fmt.Sprintf("%v = ?", column))
		if i != len(columns)-1 {
			sqlBuilder.WriteString(", ")
		}
	}
	sqlBuilder.WriteString(fmt.Sprintf(" WHERE %v = ?", primaryKeyColumnName))
	return sqlBuilder
}

func generateParameterPlaceHolders(paramsLength int) string {
	slice := make([]string, paramsLength)
	for i := range slice {
		slice[i] = "?"
	}

	return strings.Join(slice, ", ")
}

func getSQLType(fieldType string) string {
	switch fieldType {
	case "string":
		return "TEXT"
	case "int":
		return "INTEGER"
	default:
		panic("Unrecognized type: " + fieldType)
	}
}
