package rest

import (
	"github.com/emicklei/go-restful"
	"strconv"
	"gopkg.in/gorp.v1"
	"database/sql"
)


// Thread is a model
type Thread struct {
	Id        int64    `db:"id" json:"id"`
	Slug      string `db:"slug" json:"slug"`
	IsClosed  bool   `db:"state_is_closed" json:"isClosed"`
	Title     string `db:"title" json:"title"`

	Posts     int    `json:"posts"`

	*Message
}



func threadById(id int, db *gorp.DbMap) (*Thread, error) {
	thread := new(Thread)
	err := db.SelectOne(thread, "SELECT * FROM thread WHERE id = ?", id)
	msg, err := getMessageById(id, db)
	thread.Message = msg;
	// get likes
	return thread, err
}



func (api *RestApi) putThreadClose(request *restful.Request, response *restful.Response) {
	var params struct {
		Thread int `json:"thread"`
	}
	request.ReadEntity(&params)
	_, err := api.DbMap.Exec("UPDATE thread SET isClosed = true WHERE id = ?", params.Thread)
	if err != nil {
		response.WriteEntity(createResponse(API_QUERY_INVALID, err.Error()))
		return
	}
	response.WriteEntity(createResponse(API_STATUS_OK, params))
}

func (api *RestApi) threadPostCreate(request *restful.Request, response *restful.Response) {
	var thread Thread
	request.ReadEntity(&thread)
	result, err := thread.Message.InsertIntoDb(&api.DbMap)
	if err != nil { pnh(response, API_QUERY_INVALID, err); return }
	thread.Id, _ = result.LastInsertId()
	_, err = api.DbMap.Exec("INSERT INTO Thread (id, state_is_closed, title, slug) VALUES (?, ?, ?, ?)",
					thread.Id, thread.IsClosed,  thread.Title, thread.Slug)
	if err != nil { pnh(response, API_QUERY_INVALID, err); return }
	response.WriteEntity(createResponse(API_STATUS_OK, thread))
}

func (api *RestApi) threadGetDetails(request *restful.Request, response *restful.Response) {
	thread_id, _ := strconv.Atoi(request.QueryParameter("thread"))
	thread, err:= threadById(thread_id, &api.DbMap)
	if err != nil {

	}

	for _, entity := range request.Request.URL.Query()["related"] {
		if entity == "user" {
			thread.User, _ = userByEmail(thread.User.(string), &api.DbMap)
		} else if entity == "forum" {
			thread.Forum, _ = forumByShortName(thread.Forum.(string), &api.DbMap)
		} else {
			response.WriteEntity(createResponse(API_QUERY_INVALID, "Bad request"))
			return
		}
	}
	response.WriteEntity(createResponse(API_STATUS_OK, thread))
}

func (api *RestApi) threadGetList(request *restful.Request, response *restful.Response) {
	var ( queryColumn string; queryParameter string; )
	if user := request.QueryParameter("user"); user != "" {
		queryColumn = "user"; queryParameter = user
	} else { queryColumn = "forum"; queryParameter = request.QueryParameter("forum") }
	var threads []Thread
	_, err := execListQuery(request, &threads, &api.DbMap, "*", "Thread", queryColumn, queryParameter,
			"since", "date", "date", false)
	if err != nil {
		pnh(response, API_QUERY_INVALID, err)
	} else {
		response.WriteEntity(createResponse(API_STATUS_OK, threads))
	}
}

func (api *RestApi) threadGetListPosts(request *restful.Request, response *restful.Response) {
	query := "SELECT * FROM post WHERE thread = " + request.QueryParameter("thread")
	if since := request.QueryParameter("since"); since != "" {
		query += " AND date >= " + "\"" + since + "\""
	}
	var posts []Post
	_, err := execListQuery(request, &posts, &api.DbMap, "*", "Post", "thread",
				request.QueryParameter("thread"), "since", "date", "date", false)
	if err != nil {
		pnh(response, API_QUERY_INVALID, err)
	} else {
		response.WriteEntity(createResponse(API_STATUS_OK, posts))
	}
}

func (api *RestApi) threadOpen(request *restful.Request, response *restful.Response) {
	var params struct {
		Thread int `json:"thread"`
	}
	request.ReadEntity(&params)
	api.DbMap.Exec("UPDATE Thread SET state_is_closed = false WHERE id = ?", params.Thread)
	response.WriteEntity(createResponse(API_STATUS_OK, params))
}

func threadSetDeletedById(id int, deleted bool, db *gorp.DbMap) (sql.Result, error) {
	return db.Exec("UPDATE Message SET status_is_deleted = ? WHERE id = ? " +
			"OR IN (SELECT id FROM POST WHERE thread_id = ?)", deleted, id, id)
}

func threadSetDeleted(request *restful.Request, response *restful.Response, db *gorp.DbMap, delete bool) {
	var params struct {
		Thread int `json:"thread"`
	}
	request.ReadEntity(&params)
	_, err := threadSetDeletedById(params.Thread, true, db)
	if err != nil {
		pnh(response, API_QUERY_INVALID, err)
	} else { response.WriteEntity(createResponse(API_STATUS_OK, params)) }
}

func (api *RestApi) threadPostRemove(request *restful.Request, response *restful.Response) {
	threadSetDeleted(request, response, &api.DbMap, true)
}

func (api *RestApi) threadPostRestore(request *restful.Request, response *restful.Response) {
	threadSetDeleted(request, response, &api.DbMap, false)
}

func (api *RestApi) threadSubscribe(request *restful.Request, response *restful.Response) {
	var params struct {
		User   string `json:"user"`
		Thread int    `json:"thread"`
	}
	request.ReadEntity(&params)
	api.DbMap.Exec("INSERT INTO UserSubscription (user, thread_id) VALUES (?, ?)",
			params.User, params.Thread)
	response.WriteEntity(createResponse(API_STATUS_OK, params))
}

func (api *RestApi) threadUnsubscribe(request *restful.Request, response *restful.Response) {
	var params struct {
		User   string `json:"user"`
		Thread int    `json:"thread"`
	}
	request.ReadEntity(&params)
	api.DbMap.Exec("DELETE FROM usersubscription WHERE user = ? AND thread_id = ?",
			params.User, params.Thread)
	response.WriteEntity(createResponse(API_STATUS_OK, params))
}

func (api *RestApi) threadPostUpdate(request *restful.Request, response *restful.Response) {
	var params struct {
		Message string `json:"message"`
		Slug    string `json:"slug"`
		Thread  int    `json:"thread"`
	}
	request.ReadEntity(&params)
	api.DbMap.Exec("UPDATE thread SET message = ?, slug = ? WHERE id = ?",
			params.Message, params.Slug, params.Thread)
	thread, _ := threadById(params.Thread, &api.DbMap)
	response.WriteEntity(createResponse(API_STATUS_OK, thread))
}

func (api *RestApi) threadPostVote(request *restful.Request, response *restful.Response) {
	var params struct {
		User	string	`json:"user"`
		Vote  	int	`json:"vote"`
		Thread	int	`json:"thread"`
	}
	request.ReadEntity(&params)
	var is_like bool = params.Vote > 0;
	err := voteOnMessageById(params.Thread, params.User, is_like, &api.DbMap)
	if err != nil {
		response.WriteEntity(createResponse(API_QUERY_INVALID, err.Error()))
		return
	}
	thread, _ := threadById(params.Thread, &api.DbMap)
	response.WriteEntity(createResponse(API_STATUS_OK, thread))
}
