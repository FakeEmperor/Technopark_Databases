package rest

import (
	"github.com/emicklei/go-restful"
	"gopkg.in/gorp.v1"
	"log"
)


type Forum struct {
	Id        int64		`db:"id" json:"id"`
	ShortName string	`db:"short_name" json:"short_name"`
	Name      string	`db:"name" json:"name"`
	User      interface{}	`db:"user" json:"user"` /* Can be string or interface{} */
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
	api.Container	.Add(ws)
}

func forumByShortName(shortName string, db *gorp.DbMap) (*Forum, error ) {
	forum := new(Forum)
	err := db.SelectOne(&forum, "SELECT * FROM forum WHERE short_name = ?", shortName)
	return forum, err
}

func (api *RestApi) forumPostCreate(request *restful.Request, response *restful.Response) {
	var forum Forum
	request.ReadEntity(&forum)
	log.Printf("Got from request:\n %+v", forum)
	result, err := api.DbMap.Exec("INSERT INTO Forum (name, short_name, user) VALUES (?, ?, ?)", forum.Name, forum.ShortName, forum.User)
	if err != nil {
		response.WriteEntity(createResponse(API_QUERY_INVALID, err.Error()))
	} else {
		forum.Id, _ = result.LastInsertId()
		response.WriteEntity(createResponse(API_STATUS_OK, forum))
	}
}


func (api *RestApi) forumGetDetails(request *restful.Request, response *restful.Response) {
	forum, err := forumByShortName(request.QueryParameter("forum"), &api.DbMap)
	if err != nil {
		response.WriteEntity(createResponse(API_QUERY_INVALID, err.Error()))
		return
	}
	for _, entity := range request.Request.URL.Query()["related"] {
		if entity == "user" {
			log.Printf("user string is: %s", forum.User)
			user, _ := userByEmail(string(forum.User.([]uint8)), &api.DbMap)
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
	execListQuery(request,  &posts , &api.DbMap, "*", "Post", "forum_id",
			request.QueryParameter("forum"), "since", "date", "date", false);
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
	for _, post := range posts {
		if relatedUser {
			post.User, _ = userByEmail(post.User.(string), &api.DbMap)
		}
		if relatedForum {
			post.Forum, _ = forumByShortName(post.Forum.(string), &api.DbMap)
		}
		if relatedThread {
			 post.Thread, _ = threadById(post.Thread.(int), &api.DbMap)
		}
	}
	/* MAGIC ENDS HERE */
	response.WriteEntity(createResponse(0, posts))
}
/*
func (api *RestApi) forumGetListThreads(request *restful.Request, response *restful.Response) {
	related, relatedUser, relatedForum := context.Request.URL.Query()["related"], false, false
	for _, entity := range related {
		if entity == "user" {
			relatedUser = true
		} else if entity == "forum" {
			relatedForum = true
		}
	}
	query := "SELECT * FROM thread WHERE forum = " + "\"" + request.QueryParameter("forum") + "\""
	if since := request.QueryParameter("since"); since != "" {
		query += " AND date >= " + "\"" + since + "\""
	}
	query += " ORDER BY date " + context.DefaultQuery("order", "DESC")
	if limit := request.QueryParameter("limit"); limit != "" {
		query += " LIMIT " + limit
	}
	var threads []Thread
	api.DbMap.Select(&threads, query)
	response := make([]gin.H, len(threads))
	for index, thread := range threads {
		response[index] = gin.H{"date": thread.Date, "dislikes": thread.Dislikes, "forum": thread.Forum, "id": thread.ID, "isClosed": thread.IsClosed, "isDeleted": thread.IsDeleted, "likes": thread.Likes, "message": thread.Message, "points": thread.Points, "posts": thread.Posts, "slug": thread.Slug, "title": thread.Title, "user": thread.User}
		if relatedUser {
			response[index]["user"] = db.userByEmail(response[index]["user"].(string))
		}
		if relatedForum {
			response[index]["forum"] = db.forumByShortName(response[index]["forum"].(string))
		}
	}
	context.JSON(200, gin.H{"code": 0, "response": response})
}
*/
func (api *RestApi) forumGetListUsers(request *restful.Request, response *restful.Response) {
	var emails []string;
	_, err := execListQuery(request, &emails, &api.DbMap, "email", "User", "email",
		"(SELECT DISTINCT user FROM Message WHERE forum_id = \"" + request.QueryParameter("forum") + "\")",
		"since_id", "id", "name", true)
	if err != nil { pnh(response, API_QUERY_INVALID, err); return }
	users := make([]FilledUser, len(emails))
	for index, email := range emails {
		tmp, _ := userByEmail(email, &api.DbMap);
		users[index] = *tmp
	}
	response.WriteEntity(createResponse(0, users))
}


