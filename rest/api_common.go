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
		TABLE_FORUM, TABLE_POST_RATES, TABLE_POST,
		TABLE_USER, TABLE_FOLLOWERS,
		TABLE_SUBS, TABLE_THREAD_RATES, TABLE_THREAD,
		TABLE_POST_USERS,
	}
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

	if hasError {
		log.Print("[ !!! ] Error: during clear there was an error!")
		msg := "Errors: "
		for _, err := range errs {
			if err != nil { msg+=" "+ err.Error() +", \n" 	}
		}
		pnh(response, API_UNKNOWN_ERROR, err)
	} else {
		err = tr.Commit()
		if err != nil {
			pnh(response, API_UNKNOWN_ERROR, err)
		}
		response.WriteEntity(createResponse(API_STATUS_OK, "OK"));
	}
}

func (api *RestApi) commonGetStatus(request *restful.Request, response *restful.Response) {
	status:= Status{}
	var err error
	tables := map[string]*int{
		TABLE_USER : &status.Users,
		TABLE_FORUM : &status.Forums,
		TABLE_POST : &status.Posts,
		TABLE_THREAD : &status.Threads,
	}
	for name, result_addr := range tables {
		if DIRTY_USE_ESTIMATION {
			err = api.Db.QueryRow("SELECT TABLE_ROWS FROM information_schema.TABLES WHERE table_name = ?", name).Scan(result_addr)
		} else {
			err = api.Db.QueryRow("SELECT COUNT(*) FROM " + name).Scan(result_addr)
		}
		if err != nil {
			pnh(response, API_UNKNOWN_ERROR, err)
			return;
		}
	}
	response.WriteEntity(*createResponse(API_STATUS_OK, status))

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