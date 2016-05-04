package rest

import (
	"github.com/emicklei/go-restful"
	"log"
)

type Status struct {
	Users   int	`json:"users"`
	Threads int	`json:"threads"`
	Forums  int	`json:"forums"`
	Posts   int	`json:"posts"`
}


func (api *RestApi) commonPostClear(request *restful.Request, response *restful.Response) {
	truncate_tables := []string{
		"UserSubscription",  "UserFollowers", "UserMessageRate",
		"Post", "Thread", "Message", "Forum", "User" }
	log.Print("[ D ] Requested CLEAR...")
	errs := make([]error, len(truncate_tables))
	hasError := false
	tr, err := api.DbSqlx.Begin()
	if err != nil { pnh(response, API_UNKNOWN_ERROR, err); return; }
	for index, table := range truncate_tables {
		_, err := tr.Exec("DELETE FROM " + table)
		if (err != nil) { hasError = true }
		errs[index] = err
	}
	tr.Commit()
	if hasError {
		log.Print("[ !!! ] Error: during clear there was an error!")
		msg := "Errors: "
		for _, err := range errs {
			if err != nil { msg+=" "+ err.Error() +", \n" 	}
		}
		response.WriteEntity(createResponse(API_UNKNOWN_ERROR, msg))

	} else {
		response.WriteEntity(createResponse(API_STATUS_OK, "OK"));
	}
}

func (api *RestApi) commonGetStatus(request *restful.Request, response *restful.Response) {
	status:= Status{}
	var err error
	err = api.Db.QueryRow("SELECT count(id) FROM user").Scan(&status.Users)
	err = api.Db.QueryRow("SELECT count(id) FROM thread").Scan(&status.Threads)
	err = api.Db.QueryRow("SELECT count(id) FROM forum").Scan(&status.Forums)
	err = api.Db.QueryRow("SELECT count(id) FROM post").Scan(&status.Posts)
	if err != nil {
		response.WriteEntity(*createResponse(API_QUERY_INVALID, err.Error()))
	} else {
		response.WriteEntity(*createResponse(API_STATUS_OK, status))
	}
}


func (api *RestApi) registerCommonApi() {
	ws := new(restful.WebService)
	ws.
		Path("/db/api/").
		Consumes(restful.MIME_JSON).
		Produces(restful.MIME_JSON)
	ws.Route(ws.GET("/status").To(api.commonGetStatus))
	ws.Route(ws.POST("/clear").To(api.commonPostClear))

	api.Container.Add(ws)
}