package rest

import (
	"github.com/emicklei/go-restful"
	"github.com/jmoiron/sqlx"
	"log"
	"strings"
	"strconv"
)


// Thread is a model
type Thread struct {
	Slug      string	`db:"slug" json:"slug"`
	IsClosed  bool		`db:"status_is_closed" json:"isClosed"`
	Title     string	`db:"title" json:"title"`

	Posts     int64		`db:"calc_post_count" json:"posts"`

	*Message
}


// ---- static functions for Thread
func threadById(id int64, db *sqlx.DB) (*Thread, error) {
	var thread Thread;
	err := db.Get(&thread, "SELECT * FROM "+ TABLE_THREAD +" WHERE id = ?", id)
	// TODO: Get points
	if err == nil {
		backToUTF(&thread.Forum, &thread.User)
	}
	return &thread, err
}


// -------- ^^ END: THREAD ^^ -------

func (api *RestApi) registerThreadApi() {
	ws := new(restful.WebService)
	ws.
	Path("/db/api/thread/").
	Consumes(restful.MIME_JSON).
	Produces(restful.MIME_JSON)
	ws.Route(ws.GET("/details").To(api.threadGetDetails))
	ws.Route(ws.POST("/create").To(api.threadPostCreate))
	ws.Route(ws.POST("/update").To(api.threadPostUpdate))

	ws.Route(ws.POST("/open").To(api.threadPostOpen))
	ws.Route(ws.POST("/close").To(api.threadPostClose))
	ws.Route(ws.POST("/restore").To(api.threadPostRestore))
	ws.Route(ws.POST("/remove").To(api.threadPostRemove))

	ws.Route(ws.GET("/list").To(api.threadGetList))
	ws.Route(ws.GET("/listPosts").To(api.threadGetListPosts))

	ws.Route(ws.POST("/subscribe").To(api.threadPostSubscribe))
	ws.Route(ws.POST("/unsubscribe").To(api.threadPostUnsubscribe))
	ws.Route(ws.POST("/vote").To(api.threadPostVote))

	api.Container	.Add(ws)
}




func (api *RestApi) threadPostCreate(request *restful.Request, response *restful.Response) {
	var thread Thread
	request.ReadEntity(&thread)
	log.Printf("[ * ] [ THREAD CREATE ] Got thread info: %+v", thread)
	err := api.DbSqlx.Get(
		&thread.Id,
		"CALL thread_create (?, ?, ?, ?,   ?, ?, ?, ?)",
		thread.User.(string), thread.Title, thread.Slug,
		thread.Forum.(string), thread.Date,
		thread.Message.Message,
		thread.IsClosed, thread.IsDeleted,
	)
	if err != nil { pnh(response, API_QUERY_INVALID, err); return }
	response.WriteEntity(createResponse(API_STATUS_OK, thread))
}

func (api *RestApi) threadGetDetails(request *restful.Request, response *restful.Response) {
	thread_id, _ := toInt64(request.QueryParameter("thread"))
	thread, err:= threadById(thread_id, api.DbSqlx)
	if err != nil {
		pnh(response, API_NOT_FOUND, err );
		return;
	}

	for _, entity := range request.Request.URL.Query()["related"] {
		if entity == "user" && thread.User != nil  {
			thread.User, _ = userByEmail(thread.User.(string), api.DbSqlx)
		} else if entity == "forum" && thread.Forum != nil {
			thread.Forum, _ = forumByShortName(thread.Forum.(string), api.DbSqlx)
		} else {
			response.WriteEntity(createResponse(API_QUERY_INVALID, "Bad request"))
			return
		}
	}
	log.Printf("[ * ] [ THREAD DETAILS ] Got thread: %+v", thread)
	response.WriteEntity(createResponse(API_STATUS_OK, thread))
}

func (api *RestApi) threadGetList(request *restful.Request, response *restful.Response) {
	var ( queryColumn string; queryParameter string; )
	if user := request.QueryParameter("user"); user != "" {
		queryColumn = "user"; queryParameter = user
	} else { queryColumn = "forum"; queryParameter = request.QueryParameter("forum") }
	var threads []Thread
	_, err := execListQuery(
		ExecListParams{
			BuildListParams: BuildListParams{
				request: request,db: api.DbSqlx,
				selectWhat: "*", selectFromWhat: TABLE_THREAD, selectWhereColumn: queryColumn,
				selectWhereWhat: queryParameter, selectWhereIsInnerSelect: false,
				sinceParamName: "since", sinceByWhat: "date", orderByWhat: "date",
				joinEnabled: false,
				limitEnabled: true,
			},
			resultContainer: &threads,
		})
	if err != nil {
		pnh(response, API_QUERY_INVALID, err); return;
	} else {
		if threads == nil {
			threads = []Thread{}
		} else {
			for i, _ := range threads {
				backToUTF(&threads[i].Forum, &threads[i].User)
			}
		}
		response.WriteEntity(createResponse(API_STATUS_OK, threads))
	}
}



//assumes posts are ordered
func _buildNestedPostList(posts []Post) ([]NestedPost) {
	if len(posts) == 0 {
		return []NestedPost{}
	}
	mask_stack := NewStack()
	mask_match := false
	nposts := []NestedPost{}
	last := -1
	for _, post := range posts {
		mask_match = false
		for !mask_match {
			first, _ := mask_stack.First()
			if first != nil && strings.HasPrefix(post.MPath.String, first.(string)) {
				mask_match = true
				nposts[last].childs = append(nposts[last].childs, *NewNestedPost(&post))
				mask_stack.Push(post.MPath.String)
			} else {
				if len(mask_stack.s) == 0 {
					mask_stack.Push(post.MPath.String)
					mask_match = true
					nposts = append(nposts, *NewNestedPost(&post))
					last += 1
				} else {
					mask_stack.Pop()
				}
			}
		}
	}
	return nposts
}

func (api *RestApi) threadGetListPosts_Flat(BaseParams  ExecListParams) (error) {
	_, err := execListQuery( BaseParams )
	return err
}


func _getParentPosts(parents *[]int64, BaseParams ExecListParams) (error) {
	GetListParams := BaseParams;
	// todo: Make WHERE with AND
	GetListParams.selectWhereColumn = "parent_id IS NULL AND thread_id";
	GetListParams.resultContainer = parents; GetListParams.selectWhat = "id";
	_, err := execListQuery(GetListParams)
	return err
}


func (api *RestApi) threadGetListPosts_Tree(BaseParams  ExecListParams, LimitIsRecursive bool) (error){
	var err error;
	var parents []int64;
	var posts *[]Post = BaseParams.resultContainer.(*[]Post)
	//how many posts we can fetch total
	globalLimit, err := toInt64(BaseParams.request.QueryParameter("limit"));
	// turn on atomic
	BaseParams.operationIsTransaction = true;
	BaseParams.operationTransaction, err = api.DbSqlx.Beginx()

	if err == nil {
		err = _getParentPosts(&parents, BaseParams)
		if err == nil {
			var tmp_posts []Post;
			BaseParams.resultContainer = &tmp_posts;
			BaseParams.orderByWhat = "material_path"
			BaseParams.selectWhereColumn = "material_path"
			BaseParams.selectWhereCustomOp = "LIKE";
			BaseParams.sinceParamName = ""
			BaseParams.orderOverrideOrder = "ASC"

			BaseParams.limitOverrideEnabled = true;
			if LimitIsRecursive {
				BaseParams.limitEnabled = false; //as we are not counting parent node in LIMIT, but it gets selected in query
			}
			BaseParams.limitOverrideValue = globalLimit
			remains := globalLimit
			for _, parent := range parents {
				BaseParams.selectWhereWhat = MaterializedPathTerm(parent)+"%"
				_, err = execListQuery(BaseParams)
				if err != nil {
					log.Printf("[ E ] [api_thread::threadGetListPosts] Error getting child posts: %s",
						err.Error())
					break;
				}
				*posts = append(*posts, tmp_posts...)
				tmp_posts = nil
				if !LimitIsRecursive {
					remains = globalLimit - int64(len(*posts))
					BaseParams.limitOverrideValue = remains
				}
				if remains == 0 { break; }
			}
		}

		// if shit hits the fan
		defer func() {
			if err != nil {
				BaseParams.operationTransaction.Rollback()
			}
		} ();
		BaseParams.operationTransaction.Commit()
	}


	return err
}

func (api *RestApi) threadGetListPosts(request *restful.Request, response *restful.Response) {
	var posts []Post
	result_is_nested_list := false;
	var sort = strings.ToLower(request.QueryParameter("sort"))
	log.Printf("[ L ] Listing Posts... [sorting=%s]", sort)
	var err error;
	thread_id, err := strconv.Atoi(request.QueryParameter("thread"))
	if err!=nil {
		pnh(response, API_QUERY_INVALID, err)
	}
	BaseParams := ExecListParams {
		BuildListParams: BuildListParams{
			request: request, db: api.DbSqlx,
			selectWhat: "*", selectFromWhat: TABLE_POST, selectWhereColumn: "thread_id",
			selectWhereWhat: thread_id,
			sinceParamName: "since", sinceByWhat: "date", orderByWhat: "date",
			joinEnabled: false,
			limitEnabled: true,
		},
		resultContainer: &posts,
	}
	if sort == "flat" || sort == "" {
		err = api.threadGetListPosts_Flat(BaseParams)
	} else if sort == "tree" {
		//result_is_tree = true
		err = api.threadGetListPosts_Tree(BaseParams, false)
	} else { // sort == "parent_tree"
		//result_is_nested_list = true;
		err = api.threadGetListPosts_Tree(BaseParams, true)
	}

	if err != nil {
		pnh(response, API_QUERY_INVALID, err)
	} else {
		if posts == nil { posts = []Post{} } else {
			for _, post := range posts {
				backToUTF(&post.Forum , &post.User)
			}
		}
		if result_is_nested_list {
			response.WriteEntity(createResponse(API_STATUS_OK, _buildNestedPostList(posts)))
		} else {
			response.WriteEntity(createResponse(API_STATUS_OK, posts))
		}
	}
}

func threadSetDeletedById(id int64, deleted bool, db *sqlx.DB) (error) {
	_, err := db.Query("CALL thread_delete_restore(?,?)", id, deleted);

	return err;
}


func threadSetDeleted(request *restful.Request, response *restful.Response, db *sqlx.DB, deleted bool) {
	var params struct {
		Thread int64 `json:"thread"`
	}
	request.ReadEntity(&params)
	err := threadSetDeletedById(params.Thread, deleted, db)
	if err != nil {
		pnh(response, API_QUERY_INVALID, err)
	} else { response.WriteEntity(createResponse(API_STATUS_OK, params)) }
}

func (api *RestApi) threadPostRemove(request *restful.Request, response *restful.Response) {
	threadSetDeleted(request, response, api.DbSqlx, true)
}

func (api *RestApi) threadPostRestore(request *restful.Request, response *restful.Response) {
	threadSetDeleted(request, response, api.DbSqlx, false)
}

func (api *RestApi) threadPostSubscribe(request *restful.Request, response *restful.Response) {
	var params struct {
		User   string `json:"user"`
		Thread int    `json:"thread"`
	}
	request.ReadEntity(&params)
	log.Printf("[ L ] [ THREAD SUBSCRIBE ]: %+v", params)
	_, err := api.DbSqlx.Query("CALL thread_subscribe(?,?)",
			params.User, params.Thread)
	if err != nil {
		pnh(response, API_QUERY_INVALID, err)
	} else {
		response.WriteEntity(createResponse(API_STATUS_OK, params))
	}
}

func (api *RestApi) threadPostUnsubscribe(request *restful.Request, response *restful.Response) {
	var params struct {
		User   string `json:"user"`
		Thread int    `json:"thread"`
	}
	request.ReadEntity(&params)
	log.Printf("[ L ] [ THREAD UNSUBSCRIBE ]: %+v", params)
	_, err := api.DbSqlx.Query("CALL thread_unsubscribe(?,?)",
			params.User, params.Thread)
	if err != nil {
		pnh(response, API_QUERY_INVALID, err)
	} else {
		response.WriteEntity(createResponse(API_STATUS_OK, params))
	}
}

func (api *RestApi) threadPostUpdate(request *restful.Request, response *restful.Response) {
	var params struct {
		Message string `json:"message"`
		Slug    string `json:"slug"`
		Thread  int64    `json:"thread"`
	}
	request.ReadEntity(&params)
	_, err := api.DbSqlx.Exec("UPDATE "+TABLE_THREAD+" SET message = ?, slug = ? WHERE id = ?",
			params.Message, params.Slug, params.Thread)
	if err != nil {
		pnh(response, API_UNKNOWN_ERROR, err);
		return ;
	}
	thread, _ := threadById(params.Thread, api.DbSqlx)
	response.WriteEntity(createResponse(API_STATUS_OK, thread))
}


func threadOpenClose(openTrueCloseFalse bool, db *sqlx.DB, request *restful.Request, response *restful.Response) {
	var params struct {
		Thread int `json:"thread"`
	}
	request.ReadEntity(&params)
	result, err := db.Exec(
		"UPDATE "+TABLE_THREAD+" SET status_is_closed = ? WHERE id = ?",
		!openTrueCloseFalse, params.Thread)
	if err != nil {
		stat := API_UNKNOWN_ERROR
		rows, _ := result.RowsAffected()
		if rows == 0 { stat = API_NOT_FOUND }
		pnh(response, stat, err); return
	} else { response.WriteEntity(createResponse(API_STATUS_OK, params)) }
}

func (api *RestApi) threadPostOpen(request *restful.Request, response *restful.Response) {
	threadOpenClose(true, api.DbSqlx, request, response)

}

func (api *RestApi) threadPostClose(request *restful.Request, response *restful.Response) {
	threadOpenClose(false, api.DbSqlx, request, response)
}

// TODO: Change it
func (api *RestApi) threadPostVote(request *restful.Request, response *restful.Response) {
	var params struct {
		User	string	`json:"user"`
		Vote  	int	`json:"vote"`
		Thread	int64	`json:"thread"`
	}
	request.ReadEntity(&params)
	var is_like bool = params.Vote > 0;
	_, err := api.DbSqlx.Query("CALL thread_vote(?,?)",params.Thread, is_like)
	if err != nil {
		response.WriteEntity(createResponse(API_QUERY_INVALID, err.Error()))
		return
	}
	thread, _ := threadById(params.Thread, api.DbSqlx)
	response.WriteEntity(createResponse(API_STATUS_OK, thread))
}
