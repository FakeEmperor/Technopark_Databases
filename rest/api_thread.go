package rest

import (
	"github.com/emicklei/go-restful"
	"database/sql"
	"github.com/jmoiron/sqlx"
	"log"
	"strings"
)


// Thread is a model
type Thread struct {
	Slug      string	`db:"slug" json:"slug"`
	IsClosed  bool		`db:"status_is_closed" json:"isClosed"`
	Title     string	`db:"title" json:"title"`

	Posts     int64		`json:"posts"`

	*Message
}

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

func (t *Thread) getPostsCount(db *sqlx.DB) (error) {
	log.Printf("[ L ] Getting Thread (%d) post count...", t.Id)
	return db.Get( &t.Posts, "SELECT COUNT(Post.id) FROM Post JOIN Message ON Post.id = Message.id AND "+
	"Message.status_is_deleted = 0 WHERE thread_id = ? ", t.Id )
}

func threadById(id int64, db *sqlx.DB) (*Thread, error) {
	thread := new(Thread)
	err := db.Get(thread, "SELECT * FROM Thread WHERE id = ?", id)
	msg, err := getMessageById(id, db)
	thread.Message = msg;
	if err == nil { err = thread.getPostsCount(db); }
	return thread, err
}

func (api *RestApi) threadPostCreate(request *restful.Request, response *restful.Response) {
	var thread Thread
	request.ReadEntity(&thread)
	log.Printf("[ * ] [ THREAD CREATE ] Got thread info: %+v", thread)
	result, err := thread.Message.InsertIntoDb(api.DbSqlx)
	if err != nil { pnh(response, API_QUERY_INVALID, err); return }
	thread.Id, _ = result.LastInsertId()
	_, err = api.DbSqlx.Exec("INSERT INTO Thread (id, status_is_closed, title, slug) VALUES (?, ?, ?, ?)",
					thread.Id, thread.IsClosed,  thread.Title, thread.Slug)
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
			request: request, resultContainer: &threads, db: api.DbSqlx,
			selectWhat: "*", selectFromWhat: "Thread", selectWhereColumn: queryColumn,
			selectWhereWhat: queryParameter, selectWhereIsInnerSelect: false,
			sinceParamName: "since", sinceByWhat: "date", orderByWhat: "date",
			joinEnabled: true, joinTables: []string{"Message"},
			joinConditions: []string{"id"}, joinByUsingStatement: true,
			limitEnabled: true,
		})
	if err != nil {
		pnh(response, API_QUERY_INVALID, err); return;
	} else {
		if threads == nil { threads = []Thread{} } else {
			for i, _ := range threads {
				backToUTF(&threads[i].Forum, &threads[i].User)
				err = threads[i].getPostsCount(api.DbSqlx);
				if err != nil {
					pnh(response, API_UNKNOWN_ERROR, err); return;
				}
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
	BaseParams := ExecListParams {
		request: request, resultContainer: &posts, db: api.DbSqlx,
		selectWhat: "*", selectFromWhat: "Post", selectWhereColumn: "thread_id",
		selectWhereWhat: request.QueryParameter("thread"),
		sinceParamName: "since", sinceByWhat: "date", orderByWhat: "date",
		joinEnabled: true, joinTables: []string{"Message"},
		joinConditions: []string{"id"}, joinByUsingStatement: true,
		limitEnabled: true,
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
				post.getPoints(api.DbSqlx)
			}
		}
		if result_is_nested_list {
			response.WriteEntity(createResponse(API_STATUS_OK, _buildNestedPostList(posts)))
		} else {
			response.WriteEntity(createResponse(API_STATUS_OK, posts))
		}
	}
}

func threadSetDeletedById(id int64, deleted bool, db *sqlx.DB) (sql.Result, error) {
	return db.Exec("UPDATE Message SET status_is_deleted = ? WHERE id = ? " +
			"OR id IN (SELECT id FROM Post WHERE thread_id = ?)", deleted, id, id)
}

func threadSetDeleted(request *restful.Request, response *restful.Response, db *sqlx.DB, deleted bool) {
	var params struct {
		Thread int64 `json:"thread"`
	}
	request.ReadEntity(&params)
	_, err := threadSetDeletedById(params.Thread, deleted, db)
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
	api.DbSqlx.Exec("INSERT INTO UserSubscription (user, thread_id) VALUES (?, ?)",
			params.User, params.Thread)
	response.WriteEntity(createResponse(API_STATUS_OK, params))
}

func (api *RestApi) threadPostUnsubscribe(request *restful.Request, response *restful.Response) {
	var params struct {
		User   string `json:"user"`
		Thread int    `json:"thread"`
	}
	request.ReadEntity(&params)
	api.DbSqlx.Exec("DELETE FROM UserSubscription WHERE user = ? AND thread_id = ?",
			params.User, params.Thread)
	response.WriteEntity(createResponse(API_STATUS_OK, params))
}

func (api *RestApi) threadPostUpdate(request *restful.Request, response *restful.Response) {
	var params struct {
		Message string `json:"message"`
		Slug    string `json:"slug"`
		Thread  int64    `json:"thread"`
	}
	request.ReadEntity(&params)
	api.DbSqlx.Exec("UPDATE Message, Thread SET Message.message = ?, Thread.slug = ? WHERE Thread.id = ? AND Message.id = ?",
			params.Message, params.Slug, params.Thread, params.Thread)
	thread, _ := threadById(params.Thread, api.DbSqlx)
	response.WriteEntity(createResponse(API_STATUS_OK, thread))
}


func (api *RestApi) threadPostOpen(request *restful.Request, response *restful.Response) {
	var params struct {
		Thread int `json:"thread"`
	}
	request.ReadEntity(&params)
	result, err := api.DbSqlx.Exec("UPDATE Thread SET status_is_closed = false WHERE id = ?", params.Thread)
	if err != nil {
		stat := API_UNKNOWN_ERROR
		rows, _ := result.RowsAffected()
		if rows == 0 { stat = API_NOT_FOUND }
		pnh(response, stat, err); return
	} else { response.WriteEntity(createResponse(API_STATUS_OK, params)) }
}

func (api *RestApi) threadPostClose(request *restful.Request, response *restful.Response) {
	var params struct {
		Thread int `json:"thread"`
	}
	request.ReadEntity(&params)
	_, err := api.DbSqlx.Exec("UPDATE thread SET status_is_closed = true WHERE id = ?", params.Thread)
	if err != nil {
		response.WriteEntity(createResponse(API_QUERY_INVALID, err.Error()))
		return
	}
	response.WriteEntity(createResponse(API_STATUS_OK, params))
}

func (api *RestApi) threadPostVote(request *restful.Request, response *restful.Response) {
	var params struct {
		User	string	`json:"user"`
		Vote  	int	`json:"vote"`
		Thread	int64	`json:"thread"`
	}
	request.ReadEntity(&params)
	var is_like bool = params.Vote > 0;
	err := voteOnMessageById(params.Thread, is_like, api.DbSqlx)
	if err != nil {
		response.WriteEntity(createResponse(API_QUERY_INVALID, err.Error()))
		return
	}
	thread, _ := threadById(params.Thread, api.DbSqlx)
	response.WriteEntity(createResponse(API_STATUS_OK, thread))
}
