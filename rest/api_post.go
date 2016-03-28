package rest

import (
	"gopkg.in/gorp.v1"
	"github.com/emicklei/go-restful"
	"strconv"
)




// Post is a model
type Post struct {
	Id            int64			`json:"id" db:"id"`
	IsApproved    bool			`json:"isApproved" db:"state_is_approved"`
	IsEdited      bool			`json:"isEdited" db:"state_is_edited"`
	IsHighlighted bool			`json:"isHighlighted" db:"state_is_highlighted"`
	IsSpam        bool			`json:"isSpam" db:"state_is_spam"`
	Parent        int			`json:"parent" db:"parent_id"`
	Thread        interface{}		`json:"thread" db:"thread_id"`		/* Can be struct or int*/

	*Message

}

func postById(id int, db *gorp.DbMap) (*Post, error) {
	post := new(Post)
	db.SelectOne(post, "SELECT * FROM Post WHERE id = ?", id)
	msg, err := getMessageById(id, db)
	post.Message = msg;
	return post, err
}

func (api *RestApi) postCreate(request *restful.Request, response *restful.Response) {
	var post Post
	request.ReadEntity(&post)
	result, _ := post.Message.InsertIntoDb(&api.DbMap)
	post.Id, _ = result.LastInsertId();
	result, err := api.DbMap.Exec("INSERT INTO Post (isApproved, isEdited, isHighlighted, isSpam, parent, thread, )" +
		" VALUES (?, ?, ?, ?, ?, ?)", post.IsApproved, post.IsEdited, post.IsHighlighted, post.IsSpam,
						post.Parent, post.Thread)
	if err != nil { pnh(response, API_UNKNOWN_ERROR, err) } else {
		response.WriteEntity(createResponse(API_STATUS_OK, post))
	}
}

func (api *RestApi) postDetails(request *restful.Request, response *restful.Response) {
	postId, _ := strconv.Atoi(request.QueryParameter("post"))
	post, err := postById(postId, &api.DbMap);
	if  err != nil {
		pnh(response, API_NOT_FOUND, err)
	} else {
		for _, entity := range request.Request.URL.Query()["related"] {
			if entity == "user" {
				post.User, _ = userByEmail(post.User.(string), &api.DbMap)
			} else if entity == "thread" {
				post.Thread, _ = threadById(post.Thread.(int), &api.DbMap)
			} else if entity == "forum" {
				post.Forum, _ = forumByShortName(post.Forum.(string), &api.DbMap)
			}
		}
		response.WriteEntity(createResponse(API_STATUS_OK, post))
	}
}


func (api *RestApi) postList(request *restful.Request, response *restful.Response) {
	var ( queryColumn string; queryParameter string; )
	if forum := request.QueryParameter("forum"); forum != "" {
		queryColumn = "forum"; queryParameter = forum
	} else { queryColumn = "thread"; queryParameter = request.QueryParameter("thread") }
	var posts []Post
	_, err := execListQuery(request, &posts, &api.DbMap, "*", "Post", queryColumn,
			queryParameter, "since", "date", "date", false )
	if err != nil { pnh(response, API_UNKNOWN_ERROR, err) } else {
		response.WriteEntity(createResponse(API_STATUS_OK, posts))
	}


}




func (api *RestApi) postSetDeleted(request *restful.Request, response *restful.Response, deleted bool) {
	var params struct {
		Post int `json:"post"`
	}
	request.ReadEntity(&params)
	err := MessageSetDeletedById(params.Post, &api.DbMap, deleted)
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

func (api *RestApi) postUpdate(request *restful.Request, response *restful.Response) {
	var params struct {
		Post    int    `json:"post"`
		Message string `json:"message"`
	}
	request.ReadEntity(&params)
	_ , err := api.DbMap.Exec("UPDATE Message SET message = ? WHERE id = ?", params.Message, params.Post)
	if err != nil { pnh(response, API_UNKNOWN_ERROR, err) } else {
		post, _ := postById(params.Post, &api.DbMap)
		response.WriteEntity(createResponse(API_STATUS_OK, post))
	}
}

func (api *RestApi) postVote(request *restful.Request, response *restful.Response) {
	var params struct {
		Vote int `json:"vote"`
		Post int `json:"post"`
		User string `json:"user"`
	}
	request.ReadEntity(&params)
	is_like := params.Vote > 0
	voteOnMessageById(params.Vote,params.User, is_like, &api.DbMap)
	post, _ := postById(params.Post, &api.DbMap)
	response.WriteEntity(createResponse(API_STATUS_OK, post))
}
