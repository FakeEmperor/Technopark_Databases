package rest

import (
	"github.com/emicklei/go-restful"
	"encoding/json"
	"log"
	"github.com/jmoiron/sqlx"
	"database/sql"
	"strconv"
	"math"
)




// Post is a model
type Post struct {
	IsApproved    bool			`json:"isApproved" db:"status_is_approved"`
	IsEdited      bool			`json:"isEdited" db:"status_is_edited"`
	IsHighlighted bool			`json:"isHighlighted" db:"status_is_highlighted"`
	IsSpam        bool			`json:"isSpam" db:"status_is_spam"`
	Parent        *int64			`json:"parent" db:"parent_id"`
	Thread        interface{}		`json:"thread" db:"thread_id"`		/* Can be struct or int*/
	MPath	      sql.NullString		`json:"-" db:"material_path"`
	*Message
}

// post with `childs` field
type NestedPost struct {
	*Post
	childs		[]NestedPost
}

func NewNestedPost(p *Post) (*NestedPost) {
	return &NestedPost{ p, []NestedPost{}}
}


var MPATH_PADDING int = 5;
var MPATH_PADDING_POW = int64(math.Pow(10, float64(MPATH_PADDING-1)))

func MaterializedPathTerm(id int64) (string) {
	var pid string;
	pow_copy := MPATH_PADDING_POW
	if id < pow_copy && pow_copy != 1 {
		pid += "0";
		pow_copy /= 10;
	}
	pid += strconv.FormatInt(id, 10)
	return pid
}

func (post *Post) addMaterializedPath(db *sqlx.DB) (error) {
	var mp sql.NullString
	var pid string = MaterializedPathTerm(post.Id)
	if post.Parent != nil {
		err := db.Get(&mp, "SELECT material_path FROM Post WHERE id = ?", *post.Parent)
		if err != nil {
			return err
		} else {
			if !mp.Valid {
				mp.String = pid
			} else {
				mp.String += "." + pid
			}
		}
	} else {
		log.Print("[ W ] Calling addMaterializedPath() on post with no parent")
		mp.String = pid
	}
	_, err := db.Exec("UPDATE Post SET material_path = ? WHERE id = ?", mp.String, post.Id)
	return err
}



func (api *RestApi) registerPostApi() {
	ws := new(restful.WebService)
	ws.
	Path("/db/api/post/").
	Consumes(restful.MIME_JSON).
	Produces(restful.MIME_JSON)
	ws.Route(ws.GET("/details").To(api.postGetDetails))
	ws.Route(ws.POST("/create").To(api.postPostCreate))
	ws.Route(ws.POST("/update").To(api.postPostUpdate))

	ws.Route(ws.POST("/restore").To(api.postPostRestore))
	ws.Route(ws.POST("/remove").To(api.postPostRemove))

	ws.Route(ws.GET("/list").To(api.postGetList))

	ws.Route(ws.POST("/vote").To(api.postPostVote))

	api.Container.Add(ws)
}

func postById(id int64, db *sqlx.DB) (*Post, error) {
	post := new(Post)
	db.Get(post, "SELECT * FROM Post WHERE id = ?", id)
	msg, err := getMessageById(id, db)
	post.Message = msg;
	return post, err
}

func (api *RestApi) postPostCreate(request *restful.Request, response *restful.Response) {
	var post Post
	request.ReadEntity(&post)
	log.Printf("[ * ][ POST CREATE ] Got post info:\r\n %+v", post)
	result, err := post.Message.InsertIntoDb(api.DbSqlx)
	if err != nil {
		pnh(response, API_UNKNOWN_ERROR, err); return
	}
	post.Id, _ = result.LastInsertId();
	if post.Thread != nil { post.Thread, _ = post.Thread.(json.Number).Int64() }
	// THIS IS TO MAKE NULLS ON PARENT FIELD IF PARENT ID <= 0
	/////////////////
	result, err = api.DbSqlx.Exec("INSERT INTO Post (id, status_is_approved, status_is_edited, status_is_highlighted, status_is_spam, parent_id, thread_id )" +
		" VALUES (?, ?, ?, ?, ?, ?, ?)", post.Id, post.IsApproved, post.IsEdited, post.IsHighlighted,
			post.IsSpam, post.Parent, post.Thread)
	if err != nil { pnh(response, API_UNKNOWN_ERROR, err) } else {
		response.WriteEntity(createResponse(API_STATUS_OK, post))
	}
	// materialized path is created AFTER the function returns
	// actually, it is a TODO: make it run synchronously or at least synchronize it to make atomic)
	defer post.addMaterializedPath(api.DbSqlx)
}

func (api *RestApi) postGetDetails(request *restful.Request, response *restful.Response) {
	postId, _ := toInt64(request.QueryParameter("post"))
	log.Printf("[ * ][ POST DETAILS ] Getting post by id: %d", postId)
	post, err := postById(postId, api.DbSqlx);
	if  err != nil {
		pnh(response, API_NOT_FOUND, err)
	} else {
		for _, entity := range request.Request.URL.Query()["related"] {
			if entity == "user" && post.User != nil {
				post.User, _ = userByEmail(post.User.(string), api.DbSqlx)
			} else if entity == "thread" && post.Thread != nil {
				post.Thread, _ = threadById(post.Thread.(int64), api.DbSqlx)
			} else if entity == "forum" && post.Forum != nil {
				post.Forum, _ = forumByShortName(post.Forum.(string), api.DbSqlx)
			}
		}
		response.WriteEntity(createResponse(API_STATUS_OK, post))
	}
}



func (api *RestApi) postGetList(request *restful.Request, response *restful.Response) {
	var ( queryColumn string; queryParameter string; )
	if forum := request.QueryParameter("forum"); forum != "" {
		queryColumn = "forum"; queryParameter = forum
	} else { queryColumn = "thread_id"; queryParameter = request.QueryParameter("thread") }
	var posts []Post;
	_, err := execListQuery(
		ExecListParams{
			request: request, resultContainer: &posts, db: api.DbSqlx,
			selectWhat: "*", selectFromWhat: "Message", selectWhereColumn: queryColumn,
			selectWhereWhat: queryParameter, selectWhereIsInnerSelect: false,
			sinceParamName: "since", sinceByWhat: "date", orderByWhat: "date",
			joinEnabled: true, joinTables: []string{"Post"},
			joinConditions: []string{"id"}, joinByUsingStatement: true,
			limitEnabled: true} )

	if err != nil {
		pnh(response, API_UNKNOWN_ERROR, err)
	} else {
		for _, post := range posts {
			backToUTF(&post.User, &post.Forum)
		}
		if posts == nil { posts = []Post{} }
		response.WriteEntity(createResponse(API_STATUS_OK, posts))
	}
}




func (api *RestApi) postSetDeleted(request *restful.Request, response *restful.Response, deleted bool) {
	var params struct {
		Post int64 `json:"post"`
	}
	request.ReadEntity(&params)
	err := MessageSetDeletedById(params.Post, api.DbSqlx, deleted)
	if err != nil { pnh(response, API_UNKNOWN_ERROR, err) } else {
		response.WriteEntity(createResponse(API_STATUS_OK, params))
	}
}

func (api *RestApi) postPostRemove(request *restful.Request, response *restful.Response) {
	api.postSetDeleted(request, response, true)
}

func (api *RestApi) postPostRestore(request *restful.Request, response *restful.Response) {
	api.postSetDeleted(request, response, false)
}

func (api *RestApi) postPostUpdate(request *restful.Request, response *restful.Response) {
	var params struct {
		Post    int64    `json:"post"`
		Message string `json:"message"`
	}
	request.ReadEntity(&params)
	_ , err := api.DbSqlx.Exec("UPDATE Message SET message = ? WHERE id = ?", params.Message, params.Post)
	if err != nil { pnh(response, API_UNKNOWN_ERROR, err) } else {
		post, _ := postById(params.Post, api.DbSqlx)
		response.WriteEntity(createResponse(API_STATUS_OK, post))
	}
}

func (api *RestApi) postPostVote(request *restful.Request, response *restful.Response) {
	var params struct {
		Vote int `json:"vote"`
		Post int64 `json:"post"`
	}
	request.ReadEntity(&params)
	is_like := params.Vote != -1
	err := voteOnMessageById(params.Post, is_like, api.DbSqlx)
	if err != nil {
		pnh(response, API_UNKNOWN_ERROR, err)
	} else {
		post, _ := postById(params.Post, api.DbSqlx)
		response.WriteEntity(createResponse(API_STATUS_OK, post))
	}
}
