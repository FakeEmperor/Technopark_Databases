package rest

import (
	"github.com/emicklei/go-restful"
	"log"
	"gopkg.in/gorp.v1"
)


// User is a model
type User struct {
	About       *string  `db:"about" json:"about"`
	Email       string  `db:"email" json:"email"`
	Id          int64     `db:"id" json:"id"`
	IsAnonymous bool    `db:"isAnonymous" json:"status_is_anonymous"`
	Name        *string  `db:"name" json:"name"`
	Username    *string  `db:"username" json:"username"`
}

type FilledUser struct {
	About		*string		`db:"about" json:"about"`
	Email		*string		`db:"email" json:"email"`
	Id		int64		`db:"id" json:"id"`
	IsAnonymous	bool		`db:"status_is_anonymous" json:"isAnonymous"`
	Name		*string		`db:"name" json:"name"`
	Username	*string		`db:"username" json:"username"`
	Followers	[]string	`json:"followers"`
	Following	[]string	`json:"following"`
	Subscriptions	[]int		`json:"subscriptions"`
}


func (api *RestApi) registerUserApi() {
	ws := new(restful.WebService)
	ws.
	Path("/db/api/user/").
	Consumes(restful.MIME_JSON).
	Produces(restful.MIME_JSON)
	ws.Route(ws.GET("/details").To(api.userGetByEmail))
	ws.Route(ws.POST("/create").To(api.userPostCreate))
	ws.Route(ws.POST("/follow").To(api.userPostFollow))
	ws.Route(ws.POST("/unfollow").To(api.userPostUnfollow))
	ws.Route(ws.POST("/updateProfile").To(api.userPostUpdateProfile))
	ws.Route(ws.GET("/listFollowers").To(api.userGetListFollowers))
	ws.Route(ws.GET("/listFollowing").To(api.userGetListFollowing))
	ws.Route(ws.GET("/listPosts").To(api.userGetListPosts))
	api.Container	.Add(ws)
}


func userByEmail(email string, db *gorp.DbMap) (*FilledUser, error) {
	user := new(FilledUser)
	err := db.SelectOne(&user, "SELECT * FROM user WHERE email = ?", email)
	if err != nil {
		return user, err
	}
	_, err = db.Select(&user.Followers, "SELECT follower FROM userfollowers WHERE followee = ?", email)
	_, err = db.Select(&user.Following, "SELECT followee FROM userfollowers WHERE follower = ?", email)
	_, err = db.Select(&user.Subscriptions, "SELECT thread_id FROM usersubscriptions WHERE user = ?", email)
	return user, err
}

func (api *RestApi) userGetByEmail(request *restful.Request, response *restful.Response) {
	email := request.QueryParameter("user")
	log.Printf("Getting info on usr by email='%s'", email)
	user, err := userByEmail(email, &api.DbMap)
	if err != nil {
		response.WriteEntity(createResponse(API_NOT_FOUND, "User unknown"))
		return
	}
	response.WriteEntity(createResponse(API_STATUS_OK, user))
}

func (api *RestApi) userPostCreate(request *restful.Request, response *restful.Response) {
	var user User
	request.ReadEntity(&user)
	log.Printf("Got user info:\r\n %+v", user)
	result, err := api.DbMap.Exec(
		"INSERT INTO user (about, email, status_is_anonymous, name, username) VALUES (?, ?, ?, ?, ?)",
		user.About, user.Email, user.IsAnonymous, user.Name, user.Username)
	if  err != nil {
		log.Printf("[!] Error: "+err.Error())
		if len(user.Email) == 0 {
			response.WriteEntity(createResponse(API_QUERY_INVALID, "Invalid data"))
		} else {
			response.WriteEntity(createResponse(API_ALREADY_EXISTS, "Duplicate user"))
		}
		return
	} else {
		user.Id, _ = result.LastInsertId()
		response.WriteEntity(createResponse(API_STATUS_OK, user))
	}
}





func (api *RestApi) userPostFollow(request *restful.Request, response *restful.Response) {
	var params struct {
		Follower string `json:"follower"`
		Followee string `json:"followee"`
	}
	request.ReadEntity(&params)
	log.Printf("Got user info:\r\n %+v", params)
	_, err := api.DbMap.Exec("INSERT INTO userfollowers (follower, followee) VALUES (?, ?)", params.Follower, params.Followee)
	if err != nil {
		response.WriteEntity(createResponse(API_QUERY_INVALID, err.Error()))
	} else {
		response.WriteEntity(createResponse(API_STATUS_OK, params))
	}

}

func (api *RestApi) userGetListFollowers(request *restful.Request, response *restful.Response) {
	user_str := request.QueryParameter("user")
	query_str := "SELECT follower FROM userfollowers JOIN user ON user.email = followee WHERE followee =" +
			"\"" + user_str + "\""
	sinceId := request.QueryParameter("since_id");
	orderType := request.QueryParameter("order");
	limit := request.QueryParameter("limit");
	if orderType == "" { orderType = "desc" }
	if  sinceId != "" { query_str += " AND id >= " + sinceId }
	query_str += " ORDER BY follower " + orderType
	if limit != "" {
		query_str += " LIMIT " + limit
	}
	var emails []string
	api.DbMap.Select(&emails, query_str)
	users := make([]*FilledUser, len(emails))
	for index, email := range emails {
		users[index], _ = userByEmail(email, &api.DbMap)
	}
	response.WriteEntity(createResponse(API_STATUS_OK, users))
}
func (api *RestApi) userGetListFollowing(request *restful.Request, response *restful.Response) {
	query_str := "SELECT followee FROM userfollowers JOIN user ON user.email = follower WHERE follower = " +
	"\"" + request.QueryParameter("user") + "\""
	sinceId := request.QueryParameter("since_id");
	orderType := request.QueryParameter("order");
	limit := request.QueryParameter("limit");
	if orderType == "" { orderType = "desc" }
	if  sinceId != "" { query_str += " AND id >= " + sinceId }
	query_str += " ORDER BY follower " + orderType
	if limit != "" {
		query_str += " LIMIT " + limit
	}
	var emails []string
	api.DbMap.Select(&emails, query_str)
	users := make([]*FilledUser, len(emails))
	for index, email := range emails {
		users[index], _ = userByEmail(email, &api.DbMap)
	}
	response.WriteEntity(createResponse(API_STATUS_OK, users))
}


func (api *RestApi) userPostUnfollow(request *restful.Request, response *restful.Response) {
	var params struct {
		Follower string `json:"follower"`
		Followee string `json:"followee"`
	}
	request.ReadEntity(&params)
	api.DbMap.Exec("DELETE FROM userfollowers WHERE follower = ? AND followee = ?", params.Follower, params.Followee)
	usr, err := userByEmail(params.Follower, &api.DbMap);
	if err != nil {
		response.WriteEntity(createResponse(API_QUERY_INVALID, err.Error()))
	} else {
		response.WriteEntity(createResponse(API_STATUS_OK, usr))
	}
}

func (api *RestApi) userPostUpdateProfile(request *restful.Request, response *restful.Response) {
	var params struct {
		About string `json:"about"`
		User  string `json:"user"`
		Name  string `json:"name"`
	}
	request.ReadEntity(&params)
	api.DbMap.Exec("UPDATE user SET about = ?, name = ? WHERE email = ?", params.About, params.Name, params.User)
	usr, err := userByEmail(params.User, &api.DbMap);
	if err != nil {
		response.WriteEntity(createResponse(API_QUERY_INVALID, err.Error()))
	} else {
		response.WriteEntity(createResponse(API_STATUS_OK, usr))
	}
}


func (api *RestApi) userGetListPosts(request *restful.Request, response *restful.Response) {
	query := "SELECT * FROM post WHERE user = " + "\"" + request.QueryParameter("user") + "\""
	since := request.QueryParameter("since");
	orderType := request.QueryParameter("order")
	limit := request.QueryParameter("limit");
	if orderType != "desc" && orderType != "asc" { orderType = "desc"}
	if  since != "" {
		query += " AND date >= " + "\"" + since + "\""
	}
	if  limit != "" {
		query += " LIMIT " + limit
	}
	query += " ORDER BY date " + orderType;
	var posts []Post
	_, err := api.DbMap.Select(&posts, query)
	if err != nil {
		response.WriteEntity(createResponse(API_QUERY_INVALID,err.Error()))
	} else {
		response.WriteEntity(createResponse(API_STATUS_OK,posts))

	}
}

