package rest

import "database/sql"
import (
	_ "github.com/go-sql-driver/mysql"
	"github.com/emicklei/go-restful"
	//"gopkg.in/gorp.v1"
	"strings"
	"errors"
	"log"
	"github.com/jmoiron/sqlx"
	"strconv"
)


func CreateConnector() (*sql.DB, error) {
	db_connector, err:= sql.Open("mysql", "tpdb_admin:Lalka123@tcp(127.0.0.1:3306)/tpdb")
	return db_connector, err;
}

func backToUTF( uint_ptrs ... *interface{} ) {
	for _, ptr := range uint_ptrs {
		str_val := string((*ptr).([]uint8))
		*ptr = str_val
	}
}


type ExecListParams struct {
	request                  *restful.Request
	resultContainer          interface{}
	db                       *sqlx.DB

	selectWhat               string
	selectFromWhat           string
	selectWhereColumn        string
	selectWhereWhat          string
	selectWhereIsInnerSelect bool
	selectWhereCustomOp      string

	sinceParamName           string
	sinceByWhat              string

	orderByWhat              string
	orderOverrideOrder       string


	joinEnabled              bool
	joinTables               []string
	joinConditions           []string
	joinByUsingStatement     bool

	limitEnabled             bool
	limitOverrideEnabled     bool
	limitOverrideValue       int64

	operationIsTransaction   bool
	operationTransaction     *sqlx.Tx
}


func execListQuery( param ExecListParams ) ([]string, error) {
	related := param.request.Request.URL.Query()["related"]
	var op string = "= ?";
	if param.selectWhereCustomOp != "" {
		op = param.selectWhereCustomOp + " ?";
	}
	if param.selectWhereIsInnerSelect { op = "IN ( " + param.selectWhereWhat + ")" }
	from_str := param.selectFromWhat
	if param.joinEnabled {
		for index, joinTable := range param.joinTables {
			from_str += " JOIN "+ joinTable + " "
			if param.joinByUsingStatement { from_str += "USING("+param.joinConditions[index] + ") " } else {
				from_str += "ON "+param.joinConditions[index] + " "
			}
		}
	}
	query_str := " SELECT "+param.selectWhat+" FROM "+from_str+" WHERE " + param.selectWhereColumn + " "+ op
	since_str := param.request.QueryParameter(param.sinceParamName)

	limit_str := ""
	if param.limitEnabled {
		if param.limitOverrideEnabled {
			limit_str = strconv.FormatInt(param.limitOverrideValue, 10);
		} else { limit_str = param.request.QueryParameter("limit"); }
	}


	//order
	order_str := ""
	if param.orderOverrideOrder == "" { order_str = strings.ToUpper(param.request.QueryParameter("order")) } else {
		order_str = strings.ToUpper(param.orderOverrideOrder)
	}
	if order_str  == "" { order_str = ORDER_DESC }

	query_variables := []interface{}{ }

	// CASE FOR INNER SELECT
	if !param.selectWhereIsInnerSelect { query_variables = append(query_variables, param.selectWhereWhat) }

	// SINCE PARAM (ADDITIONAL CONDITION ON WHERE)
	if since_str != "" && param.sinceByWhat != "" {
		query_str += " AND "+param.sinceByWhat+" >= ?"
		query_variables = append(query_variables, since_str)
	}

	//ORDERING
	if param.orderByWhat != "" {
		if order_str != ORDER_ASC && order_str != ORDER_DESC {
			return related, errors.New("Order field from query ('"+order_str+"') is not valid")
		} else { query_str += " ORDER BY " + param.orderByWhat + " " + order_str }
	}

	//LIMITING
	if param.limitEnabled && limit_str != "" {
		query_str += " LIMIT ? "
		query_variables = append(query_variables, limit_str)
	}


	log.Printf("[ ! ] [execListQuery] Query is: %s\nWith params: %+v", query_str, query_variables)
	var err error;
	if !param.operationIsTransaction {
		err = param.db.Select(param.resultContainer, query_str, query_variables...)

	} else {
		err = param.operationTransaction.Select(param.resultContainer, query_str, query_variables...)
	}


	return related, err
}