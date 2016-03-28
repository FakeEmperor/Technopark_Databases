package rest

import "database/sql"
import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/emicklei/go-restful"
	"gopkg.in/gorp.v1"
	"strings"
	"errors"
)


func CreateConnector() (*sql.DB, error) {
	db_connector, err:= sql.Open("mysql", "tpdb_admin:Lalka123@tcp(95.213.235.64:3306)/tpdb")
	return db_connector, err;
}

func execListQuery(
	request *restful.Request, resultContainer interface{},
	db *gorp.DbMap, what string,
	fromWhat string, whereColumn string, whereWhat string,
	sinceParamName string, sinceByWhat string, orderByWhat string, innerSelectOnWhere bool) ([]string, error) {
	related := request.Request.URL.Query()["related"]
	var op string = "=";
	if innerSelectOnWhere { op = "IN" }
	query_str := " SELECT ? FROM ? WHERE ? "+ op +" ?"
	since_str := request.QueryParameter(sinceParamName)
	limit_str := request.QueryParameter("limit")
	order_str := strings.ToUpper(request.QueryParameter("order"))
	if order_str == "" { order_str = ORDER_DESC }
	query_variables := []interface{}{ what, fromWhat, whereColumn, whereWhat}
	if since_str != "" && sinceByWhat != "" {
		query_str += " AND ? >= ?"
		query_variables = append(query_variables, sinceByWhat, since_str)
	}
	if limit_str != "" {
		query_str += " LIMIT ? "
		query_variables = append(query_variables, limit_str)
	}
	if orderByWhat != "" {
		if order_str != ORDER_ASC && order_str != ORDER_DESC {
			return related, errors.New("Order field from query ('"+order_str+"') is not valid")
		} else {

			query_str += " ORDER BY ?" + orderByWhat + " " + query_str
			query_variables = append(query_variables, limit_str)
		}
	}
	_, err := db.Select(resultContainer, query_str, query_variables...)
	return related, err
}