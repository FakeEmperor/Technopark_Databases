package rest

import (
	"github.com/emicklei/go-restful"
	"log"
	"github.com/jmoiron/sqlx"
)


type Forum struct {
	Id		int64		`db:"id" json:"id"`
	ShortName	string		`db:"short_name" json:"short_name"`
	Name		string		`db:"name" json:"name"`
	User		interface{}	`db:"user" json:"user"` /* Can be string or interface{} */
}

func (api *RestApi) registerForumApi() {
	ws := new(restful.WebService)
	ws.
	Path("/db/api/forum/").
	Consumes(restful.MIME_JSON).
	Produces(restful.MIME_JSON)
	ws.Route(ws.GET("/details").To(api.forumGetDetails))
	ws.Route(ws.POST("/create").To(api.forumPostCreate))
	ws.Route(ws.GET("/listUsers").To(api.forumGetListUsers))
	ws.Route(ws.GET("/listPosts").To(api.forumGetListPosts))
	ws.Route(ws.GET("/listThreads").To(api.forumGetListThreads))

	api.Container	.Add(ws)
}

func forumByShortName(shortName string, db *sqlx.DB) (*Forum, error ) {
	forum := new(Forum)
	err := db.Get(forum, "SELECT * FROM Forum WHERE short_name = ?", shortName)
	forum.User = string(forum.User.([]uint8))
	return forum, err
}
/*
func forumById(id int64, db *sqlx.DB) (*Forum, error) {
	forum := new(Forum)
	err := db.SelectOne(&forum, "SELECT * FROM forum WHERE id = ?", id)
	return forum, err
} */

func (api *RestApi) forumPostCreate(request *restful.Request, response *restful.Response) {
	var forum Forum
	request.ReadEntity(&forum)
	log.Printf("Got from request:\n %+v", forum)
	result, err := api.DbSqlx.Exec("INSERT INTO Forum (name, short_name, user) VALUES (?, ?, ?)", forum.Name, forum.ShortName, forum.User)
	if err != nil {
		response.WriteEntity(createResponse(API_QUERY_INVALID, err.Error()))
	} else {
		forum.Id, _ = result.LastInsertId()
		response.WriteEntity(createResponse(API_STATUS_OK, forum))
	}
}


func (api *RestApi) forumGetDetails(request *restful.Request, response *restful.Response) {
	forum, err := forumByShortName(request.QueryParameter("forum"), api.DbSqlx)
	if err != nil {
		pnh(response, API_QUERY_INVALID, err)
		return
	}
	for _, entity := range request.Request.URL.Query()["related"] {
		if entity == "user" {
			log.Printf("user string is: %s", forum.User)
			user, _ := userByEmail(forum.User.(string), api.DbSqlx)
			forum.User = user;
			break;
		}
	}
	response.WriteEntity(createResponse(API_STATUS_OK, forum))
}

var (
	ORDER_DESC string = "DESC"
	ORDER_ASC string = "ASC"
)



func (api *RestApi) forumGetListPosts(request *restful.Request, response *restful.Response) {
	var posts []Post
	related, err :=
	execListQuery(
		ExecListParams{
			request: request, resultContainer: &posts, db: api.DbSqlx,
			selectWhat: "*", selectFromWhat: "Post", selectWhereColumn: "forum",
			selectWhereWhat: request.QueryParameter("forum"), selectWhereIsInnerSelect: false,
			sinceParamName: "since", sinceByWhat: "date", orderByWhat: "date",
			joinEnabled: true, joinTables: []string{ "Message" } ,
			joinConditions: []string{ "id" }, joinByUsingStatement: true,
			limitEnabled: true } );
	if err != nil {
		response.WriteEntity(createResponse(API_QUERY_INVALID, err.Error()))
		return
	}
	relatedUser, relatedThread, relatedForum := false, false, false
	for _, entity := range related {
		if entity == "user" {
			relatedUser = true
		} else if entity == "forum" {
			relatedForum = true
		} else if entity == "thread" {
			relatedThread = true
		}
	}
	for index, post := range posts {
		backToUTF(&post.User, &post.Forum)
		post.getPoints(api.DbSqlx)
		if relatedUser {
			posts[index].User, _ = userByEmail(post.User.(string), api.DbSqlx)
		}
		if relatedForum {
			posts[index].Forum, _ = forumByShortName(post.Forum.(string), api.DbSqlx)
		}
		if relatedThread {
			posts[index].Thread, _ = threadById(post.Thread.(int64), api.DbSqlx)
		}
	}
	if posts == nil { posts = []Post{} }
	response.WriteEntity(createResponse(0, posts))
}

func (api *RestApi) forumGetListThreads(request *restful.Request, response *restful.Response) {
	var threads []Thread
	related, err := execListQuery(
		ExecListParams{
			request: request, resultContainer: &threads, db: api.DbSqlx,
			selectWhat: "*", selectFromWhat: "Message", selectWhereColumn: "forum",
			selectWhereWhat: request.QueryParameter("forum"), selectWhereIsInnerSelect: false,
			sinceParamName: "since", sinceByWhat: "date", orderByWhat: "date",
			joinEnabled: true, joinTables: []string{ "Thread" },
			joinConditions: []string{ "id" }, joinByUsingStatement: true,
			limitEnabled: true })

	if err != nil { pnh(response, API_UNKNOWN_ERROR, err); return; }
	relatedUser, relatedForum := false, false
	for _, entity := range related {
		if entity == "user" {
			relatedUser = true
		} else if entity == "forum" {
			relatedForum = true
		}
	}
	for index, thread := range threads {
		backToUTF(&thread.User, &thread.Forum)
		thread.getPoints(api.DbSqlx)
		api.DbSqlx.Get(&threads[index].Posts, "SELECT COUNT(Post.id) FROM Message JOIN "+
						" Post ON Post.id = Message.id AND Message.status_is_deleted = 0 "+
						" WHERE Post.thread_id = ?", thread.Id)
		if relatedUser {
			threads[index].User, _ = userByEmail(thread.User.(string), api.DbSqlx)
		}
		if relatedForum {
			threads[index].Forum, _ = forumByShortName(thread.Forum.(string), api.DbSqlx)
		}
	}
	if threads == nil { threads = []Thread{} }
	response.WriteEntity(createResponse(0, threads))
}

func (api *RestApi) forumGetListUsers(request *restful.Request, response *restful.Response) {
	var emails []string;
	_, err := execListQuery(
		ExecListParams{
			request: request, resultContainer: &emails, db: api.DbSqlx,
			selectWhat: "email", selectFromWhat: "User", selectWhereColumn: "email",
			selectWhereWhat: "(SELECT DISTINCT user FROM Message WHERE forum = \"" + request.QueryParameter("forum") + "\")",
			selectWhereIsInnerSelect: true,
			sinceParamName: "since_id", sinceByWhat: "id", orderByWhat: "name",
			joinEnabled: false, joinTables: nil, joinConditions: nil, joinByUsingStatement: true,
			limitEnabled: true } )
	if err != nil { pnh(response, API_QUERY_INVALID, err); return }
	users := make([]FilledUser, len(emails))
	for index, email := range emails {
		tmp, _ := userByEmail(email, api.DbSqlx);
		users[index] = *tmp
	}
	response.WriteEntity(createResponse(0, users))
}


