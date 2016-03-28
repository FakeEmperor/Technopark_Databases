package rest

import (
	"log"
	"encoding/json"
)

type Response struct {
	Code     int		`json:"code"`
	Response interface{}	`json:"response"`
}


/*
◾0 — ОК,
◾1 — запрашиваемый объект не найден,
◾2 — невалидный запрос (например, не парсится json),
◾3 — некорректный запрос (семантически),
◾4 — неизвестная ошибка.
◾5 — такой юзер уже существует

*/
var (
	API_STATUS_OK int = 0
	API_NOT_FOUND int = 1
	API_QUERY_INVALID int = 2
	API_UNKNOWN_ERROR int = 4
	API_ALREADY_EXISTS int = 5
)

func createResponse(code int, response interface{}) (*Response) {
	sts := new(Response)
	sts.Code = code
	sts.Response = response
	log.Print("[*] Response creation is finished")
	s, err := json.Marshal(sts);
	log.Printf("[L] Data which is sent to client:\n%s\nERRORS: %s", s, err)
	return sts
}