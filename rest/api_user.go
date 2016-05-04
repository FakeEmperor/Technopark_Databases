package rest

import (
	"github.com/emicklei/go-restful"
	"log"
	"github.com/jmoiron/sqlx"
)


// User is a model
type User struct {
	About       *string  `db:"about" json:"about"`
	Email       string  `db:"email" json:"email"`
	Id          int64     `db:"id" json:"id"`
	IsAnonymous bool    `db:"status_is_anonymous" json:"isAnonymous"`
	Name        *string  `db:"name" json:"name"`
	Username    *string  `db:"username" json:"username"`
}

type FilledUser struct {
	*User
	Followers	[]string	`json:"followers"`
	Following	[]string	`json:"following"`
	Subscriptions	[]int64		`json:"subscriptions"`
}


func (api *RestApi) registerUserApi() {
	ws := new(restful.WebService)
	ws.
	Path("/db/api/user/").
	Consumes(restful.MIME_JSON).
	Produces(restful.MIME_JSON)
	ws.Route(ws.GET("/details").To(api.userGetDetails))
	ws.Route(ws.POST("/create").To(api.userPostCreate))
	ws.Route(ws.POST("/updateProfile").To(api.userPostUpdateProfile))

	ws.Route(ws.POST("/follow").To(api.userPostFollow))
	ws.Route(ws.POST("/unfollow").To(api.userPostUnfollow))

	ws.Route(ws.GET("/listFollowers").To(api.userGetListFollowers))
	ws.Route(ws.GET("/listFollowing").To(api.userGetListFollowing))
	ws.Route(ws.GET("/listPosts").To(api.userGetListPosts))
	api.Container	.Add(ws)
}


func userByEmail(email string, db *sqlx.DB) (*FilledUser, error) {
	var user *User = new(User);
	var fuser *FilledUser = new(FilledUser)
	err := db.Get(user, "SELECT * FROM user WHERE email = ?", email)
	if err != nil {
		return nil, err
	}
	fuser.User = user;
	err = db.Select(&fuser.Followers, "SELECT follower FROM UserFollowers WHERE followee = ?", email)
	err = db.Select(&fuser.Following, "SELECT followee FROM UserFollowers WHERE follower = ?", email)
	err = db.Select(&fuser.Subscriptions, "SELECT thread_id FROM usersubscription WHERE user = ?", email)
	if fuser.Followers == nil { fuser.Followers = []string{} }
	if fuser.Following == nil { fuser.Following = []string{} }
	if fuser.Subscriptions == nil { fuser.Subscriptions = []int64{} }
	return fuser, err
}

func (api *RestApi) userGetDetails(request *restful.Request, response *restful.Response) {
	email := request.QueryParameter("user")
	log.Printf("[*] [ USER DETAILS ] Getting info on usr by email='%s'", email)
	user, err := userByEmail(email, api.DbSqlx)
	if err != nil { pnh(response, API_NOT_FOUND, err); return }
	response.WriteEntity(createResponse(API_STATUS_OK, user))
}

func (api *RestApi) userPostCreate(request *restful.Request, response *restful.Response) {
	var user User
	request.ReadEntity(&user)
	log.Printf("[ * ][ USER CREATE ] Got user info:\r\n %+v", user)
	result, err := api.DbSqlx.Exec(
		"INSERT INTO user (about, email, status_is_anonymous, name, username) VALUES (?, ?, ?, ?, ?)",
		user.About, user.Email, user.IsAnonymous, user.Name, user.Username)
	if  err != nil {
		log.Printf("[ ! ] [ USER CREATE ] Error: "+err.Error())
		stat := API_QUERY_INVALID
		if len(user.Email) > 0 { stat = API_ALREADY_EXISTS }
		pnh(response, stat, err )
	} else {
		log.Printf("[ ok ] [ USER CREATE ] USER CREATED")
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
	_, err := api.DbSqlx.Exec("INSERT INTO userfollowers (follower, followee) VALUES (?, ?)", params.Follower, params.Followee)
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
	api.DbSqlx.Select(&emails, query_str)
	users := make([]*FilledUser, len(emails))
	for index, email := range emails {
		users[index], _ = userByEmail(email, api.DbSqlx)
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
	api.DbSqlx.Select(&emails, query_str)
	users := make([]*FilledUser, len(emails))
	for index, email := range emails {
		users[index], _ = userByEmail(email, api.DbSqlx)
	}
	response.WriteEntity(createResponse(API_STATUS_OK, users))
}


func (api *RestApi) userPostUnfollow(request *restful.Request, response *restful.Response) {
	var params struct {
		Follower string `json:"follower"`
		Followee string `json:"followee"`
	}
	request.ReadEntity(&params)
	api.DbSqlx.Exec("DELETE FROM userfollowers WHERE follower = ? AND followee = ?", params.Follower, params.Followee)
	usr, err := userByEmail(params.Follower, api.DbSqlx);
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
	api.DbSqlx.Exec("UPDATE user SET about = ?, name = ? WHERE email = ?", params.About, params.Name, params.User)
	usr, err := userByEmail(params.User, api.DbSqlx);
	if err != nil {
		response.WriteEntity(createResponse(API_QUERY_INVALID, err.Error()))
	} else {
		response.WriteEntity(createResponse(API_STATUS_OK, usr))
	}
}


func (api *RestApi) userGetListPosts(request *restful.Request, response *restful.Response) {
	var posts []Post
	_, err := execListQuery(
		ExecListParams{
			request: request, resultContainer: &posts, db: api.DbSqlx,
			selectWhat: "*", selectFromWhat: "Post", selectWhereColumn: "user",
			selectWhereWhat: request.QueryParameter("user"), selectWhereIsInnerSelect: false,
			sinceParamName: "since", sinceByWhat: "date", orderByWhat: "date",
			joinEnabled: true, joinTables: []string{"Message"},
			joinConditions: []string{"id"}, joinByUsingStatement: true,
			limitEnabled: true,
		})
	if err != nil {
		response.WriteEntity(createResponse(API_QUERY_INVALID,err.Error()))
	} else {
		if (posts == nil ) { posts = []Post{} } else {
			for _, post := range posts {
				backToUTF(&post.User, &post.Forum) //super crutch!!
				post.getPoints(api.DbSqlx)
			}
		}
		response.WriteEntity(createResponse(API_STATUS_OK,posts))

	}
}

