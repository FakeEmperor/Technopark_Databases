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
	"fmt"
)

var (
	DB_FUNCTIONAL = "tpdb_test"
	DB_PERFORMANCE = "tpdb"


	TABLE_POST = "post_merged";
	TABLE_THREAD = "thread_merged";

	TABLE_USER = "User";

	TABLE_FORUM = "Forum";

	TABLE_SUBS = "UserSubscription";
	TABLE_FOLLOWERS = "UserFollowers";
	TABLE_POST_RATES = "post_rate";
	TABLE_THREAD_RATES = "thread_rate";

	TABLE_POST_USERS = "post_users";

	DIRTY_USE_ESTIMATION = false;
)

func CreateConnector(dbname string) (*sql.DB, error) {
	db_connector, err:= sql.Open("mysql", "tpdb_admin:Lalka123@tcp(127.0.0.1:3306)/"+dbname)
	return db_connector, err;
}

func backToUTF( uint_ptrs ... *interface{} ) {
	for _, ptr := range uint_ptrs {
		if ptr != nil {
			str_val := string((*ptr).([]uint8))
			*ptr = str_val
		}

	}
}


type BuildListParams struct {
	request                  *restful.Request
	db                       *sqlx.DB

	selectWhat               string
	selectFromWhat           string
	selectWhereColumn        string
	selectWhereWhat          interface{}
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
	joinPlaceholderParams	 [][]interface{}

	limitEnabled             bool
	limitOverrideEnabled     bool
	limitOverrideValue       int64


}

type ExecListParams struct {
	BuildListParams

	resultContainer          interface{}
	operationIsTransaction   bool
	operationTransaction     *sqlx.Tx



}


func GetOrderFromQuery( request *restful.Request) (string) {
	//order
	order_str := strings.ToUpper(request.QueryParameter("order"))
	if order_str  == "" { order_str = ORDER_DESC }
	return order_str;
}

func nameSubqueryTable( subquery string, tableName string) string {
	return fmt.Sprintf("(%s) as %s", subquery, tableName)
}

func buildListQuery(param BuildListParams) (string, []interface{}, []string, error) {
	related := param.request.Request.URL.Query()["related"]
	query_variables := []interface{}{ }

	// MOST BASIC OPERATION - FROM
	from_str := param.selectFromWhat
	// CASE FOR JOIN
	if param.joinEnabled {
		jpp_len := len(param.joinPlaceholderParams)
		for index, joinTable := range param.joinTables {
			from_str += " JOIN "+ joinTable + " "
			if param.joinByUsingStatement { from_str += "USING("+param.joinConditions[index] + ") " } else {
				from_str += "ON "+param.joinConditions[index] + " "
			}

			if param.joinPlaceholderParams != nil && jpp_len > index && len(param.joinPlaceholderParams[index]) > 0 {
				query_variables = append(query_variables, param.joinPlaceholderParams[index]...)
			}
		}
	}
	// BASE SELECTION SKELETON
	query_str := " SELECT "+param.selectWhat+" FROM "+from_str;
	// IF WHERE IS ACTIVE
	if param.selectWhereColumn != "" && param.selectWhereWhat != "" {
		var op string;
		// CASE FOR INNER SELECT
		if param.selectWhereIsInnerSelect {
			op = "IN ( " + param.selectWhereWhat.(string) + ")"
		} else {
			// CASE FOR STRAIGHT WHERE
			if param.selectWhereCustomOp != "" {
				op = param.selectWhereCustomOp + " ?";
			} else {
				op = "= ?"
			}
			query_variables = append(query_variables, param.selectWhereWhat)
		}
		query_str += " WHERE " + param.selectWhereColumn + " "+ op
	}
	// CASE FOR LIMIT
	limit_str := ""
	if param.limitEnabled {
		if param.limitOverrideEnabled {
			limit_str = strconv.FormatInt(param.limitOverrideValue, 10);
		} else { limit_str = param.request.QueryParameter("limit"); }
	}

	//order
	var order_str string;
	if param.orderOverrideOrder != "" {
		order_str = param.orderOverrideOrder;
	} else { order_str = GetOrderFromQuery(param.request); }

	// CASE FOR SINCE PARAM (ADDITIONAL CONDITION ON WHERE)
	since_str := param.request.QueryParameter(param.sinceParamName)
	if since_str != "" && param.sinceByWhat != "" {
		query_str += " AND "+param.sinceByWhat+" >= ?"
		query_variables = append(query_variables, since_str)
	}

	//ORDERING
	if param.orderByWhat != "" {
		if order_str != ORDER_ASC && order_str != ORDER_DESC {
			return "", []interface{}{}, related, errors.New("Order field from query ('"+order_str+"') is not valid")
		} else { query_str += " ORDER BY " + param.orderByWhat + " " + order_str }
	}

	//LIMITING
	if param.limitEnabled && limit_str != "" {
		// if not bake - then only prepare with placeholders
		query_str += " LIMIT ? "
		query_variables = append(query_variables, limit_str)
	}


	log.Printf("[ ! ] [execListQuery] Query is: %s\nWith params: %+v", query_str, query_variables)

	return query_str, query_variables, related, nil;
}

func execListQuery( param ExecListParams ) ([]string, error) {
	query_str, query_variables , related, err := buildListQuery(param.BuildListParams);
	if err != nil {
		return related, err;
	}

	if !param.operationIsTransaction {
		err = param.db.Select(param.resultContainer, query_str, query_variables...)

	} else {
		err = param.operationTransaction.Select(param.resultContainer, query_str, query_variables...)
	}


	return related, err
}