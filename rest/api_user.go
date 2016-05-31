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

func (fuser *FilledUser) GetFollowersSubscriptions(db *sqlx.DB) (error) {
	err := db.Select(&fuser.Followers, "SELECT follower FROM UserFollowers WHERE followee = ?", fuser.Email)
	err = db.Select(&fuser.Following, "SELECT followee FROM UserFollowers WHERE follower = ?", fuser.Email)
	err = db.Select(&fuser.Subscriptions, "SELECT thread_id FROM usersubscription WHERE user = ?", fuser.Email)
	if fuser.Followers == nil { fuser.Followers = []string{} }
	if fuser.Following == nil { fuser.Following = []string{} }
	if fuser.Subscriptions == nil { fuser.Subscriptions = []int64{} }
	return err;
}



func userByEmail(email string, db *sqlx.DB) (*FilledUser, error) {
	var user *User = new(User);
	var fuser *FilledUser = new(FilledUser)
	err := db.Get(user, "SELECT * FROM user WHERE email = ?", email)
	if err != nil {
		return nil, err
	}
	fuser.User = user;
	err = fuser.GetFollowersSubscriptions(db);
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
	log.Printf("[ L ] [ USER FOLLOW ]: %+v", params)
	_, err := api.DbSqlx.Exec("INSERT INTO "+TABLE_FOLLOWERS+" (follower, followee) VALUES (?, ?)", params.Follower, params.Followee)
	if err != nil {
		pnh(response, API_UNKNOWN_ERROR, err)
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
	var users []FilledUser;
	subquery, params, _, err := buildListQuery(BuildListParams {
		db: api.DbSqlx, request: request,
		selectWhat: "followee as 'email'", selectFromWhat: "UserFollowers",
		selectWhereColumn: "follower", selectWhereWhat: request.QueryParameter("user"),
		selectWhereIsInnerSelect: false, joinEnabled: false,
	})
	if err != nil {
		pnh(response, API_UNKNOWN_ERROR, err);
		return;
	}
	execListQuery(ExecListParams{
		BuildListParams: BuildListParams{
			request: request,  db: api.DbSqlx,
			selectWhat: "*", selectFromWhat: "User", selectWhereColumn: "id",
			selectWhereCustomOp: ">=",
			selectWhereWhat: request.QueryParameter("user"), selectWhereIsInnerSelect: false,
			sinceParamName: "since_id", sinceByWhat: "id",
			joinEnabled: true, joinTables: []string{ nameSubqueryTable(subquery, "Subs")},
			joinByUsingStatement: true, joinConditions: []string{"email"},
			joinPlaceholderParams: [][]interface{}{params},
			orderByWhat: "",
			limitEnabled: true,
		},
		resultContainer: &users,
	})
	if err != nil {
		pnh(response, API_UNKNOWN_ERROR, err);
	} else {
		for i, _ := range users {
			users[i].GetFollowersSubscriptions(api.DbSqlx)
		}
		response.WriteEntity(createResponse(API_STATUS_OK, users))
	}

}


func (api *RestApi) userPostUnfollow(request *restful.Request, response *restful.Response) {
	var params struct {
		Follower string `json:"follower"`
		Followee string `json:"followee"`
	}
	request.ReadEntity(&params)
	log.Printf("[ L ] [ USER UNFOLLOW ]: %+v", params)
	_, err := api.DbSqlx.Query("CALL user_unfollow(?,?)", params.Follower, params.Followee);
	usr, err := userByEmail(params.Follower, api.DbSqlx);
	if err != nil {
		pnh(response, API_UNKNOWN_ERROR, err)
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
	log.Printf("[ L ] [ USER PROFILE UPDATE ]: %+v", params)
	_, err := api.DbSqlx.Exec("UPDATE User SET about = ?, name = ? WHERE email = ?", params.About, params.Name, params.User)
	if err != nil {
		pnh(response, API_QUERY_INVALID, err)
		return;
	}
	// usr, err := userByEmail(params.User, api.DbSqlx); // <-- HACK
	if err != nil {
		pnh(response, API_QUERY_INVALID, err)
	} else {
		response.WriteEntity(createResponse(API_STATUS_OK, params)) // <-- HACK
	}
}


func (api *RestApi) userGetListPosts(request *restful.Request, response *restful.Response) {
	var posts []Post
	_, err := execListQuery(
		ExecListParams{
			BuildListParams: BuildListParams{
				request: request,  db: api.DbSqlx,
				selectWhat: "*", selectFromWhat: TABLE_POST, selectWhereColumn: "user",
				selectWhereWhat: request.QueryParameter("user"), selectWhereIsInnerSelect: false,
				sinceParamName: "since", sinceByWhat: "date", orderByWhat: "date",
				joinEnabled: false,
				limitEnabled: true,
			},
			resultContainer: &posts,
		})
	if err != nil {
		pnh(response, API_QUERY_INVALID, err)
	} else {
		if (posts == nil ) { posts = []Post{} } else {
			for _, post := range posts {
				backToUTF(&post.User, &post.Forum) //super crutch!!
			}
		}
		response.WriteEntity(createResponse(API_STATUS_OK,posts))

	}
}

