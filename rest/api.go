package rest

import (
	"github.com/emicklei/go-restful"
	"database/sql"
	"gopkg.in/gorp.v1"
	"log"
)





type RestApi struct {
	Db *sql.DB
	DbMap gorp.DbMap

	Container *restful.Container
}

func CreateRestApi() *RestApi {
	wsContainer := restful.NewContainer()
	// Add container filter to enable CORS
	cors := restful.CrossOriginResourceSharing{
		ExposeHeaders:  []string{"X-Lalka-Header"},
		AllowedHeaders: []string{"Content-Type", "Accept"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
		CookiesAllowed: false,
		Container:      wsContainer}
	wsContainer.Filter(cors.Filter)
	// Add container filter to respond to OPTIONS
	wsContainer.Filter(wsContainer.OPTIONSFilter)

	wsContainer.Filter(globalLogging)
	var err error
	api := new(RestApi)
	api.Db, err = CreateConnector()
	api.DbMap = gorp.DbMap{Db: api.Db, Dialect: gorp.MySQLDialect{Engine: "InnoDB", Encoding: "UTF8"}}
	api.Container = wsContainer
	api.registerCommonApi()
	api.registerUserApi()
	api.registerForumApi()
	var _ = err;
	return api
}

func globalLogging(request *restful.Request, response *restful.Response, chain *restful.FilterChain) {
	log.Printf("[restful] Request: %s\n", *request )
	chain.ProcessFilter(request, response)
}


func pnh(response *restful.Response, code int, err error) {
	response.WriteEntity(createResponse(code, err.Error()))
}
