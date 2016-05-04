package main

import (
	"io"
	"log"
	"net/http"
	"./rest"
	"github.com/emicklei/go-restful"
)

// Cross-origin resource sharing (CORS) is a mechanism that allows JavaScript on a web page
// to make XMLHttpRequests to another domain, not the domain the JavaScript originated from.
//
// http://en.wikipedia.org/wiki/Cross-origin_resource_sharing
// http://enable-cors.org/server.html
//
// GET http://localhost:8080/users
//
// GET http://localhost:8080/users/1
//
// PUT http://localhost:8080/users/1
//
// DELETE http://localhost:8080/users/1
//
// OPTIONS http://localhost:8080/users/1  with Header "Origin" set to some domain and

type UserResource struct{}

func (u UserResource) RegisterTo(container *restful.Container) {
	ws := new(restful.WebService)
	ws.
	Path("/users").
	Consumes("*/*").
	Produces("*/*")

	ws.Route(ws.GET("/{user-id}").To(u.nop))
	ws.Route(ws.POST("").To(u.nop))
	ws.Route(ws.PUT("/{user-id}").To(u.nop))
	ws.Route(ws.DELETE("/{user-id}").To(u.nop))

	container.Add(ws)
}

func (u UserResource) nop(request *restful.Request, response *restful.Response) {
	io.WriteString(response.ResponseWriter, "this would be a normal response")
}


func main() {
	log.Printf("[ * ] Checking database connection...")
	conn, err := rest.CreateConnector()
	db_err_string := "[ ERROR ] Could not connect to the database.\nError:%s"
	if (err != nil) {
		log.Panic(db_err_string, err)
	} else {
		err = conn.Ping()
		if (err != nil) {
			log.Panic(db_err_string, err)
		}
	}

	log.Printf("[ * ] Loading router...")


	restApi := rest.CreateRestApi()

	log.Printf("[ * ] Beggining to listen on localhost:8080")
	server := &http.Server{Addr: ":8080", Handler: restApi.Container }
	log.Fatal(server.ListenAndServe())
}
