package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
)

func SearchServer(w http.ResponseWriter, r *http.Request) {
	//получение данных из xml. Если это у нас не выйдет, то сервис недоступен
	file, err := os.Open("dataset.xml")
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	fileInfo, err := ioutil.ReadAll(file)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	fileData := Root{}
	xml.Unmarshal(fileInfo, &fileData)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	rows := fileData.Rows

	//проверка авторизации
	token := r.Header.Get("AccessToken")
	if token != "TestToken" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	//поля из SearchRequest
	limitStr := r.URL.Query().Get("limit")

	offsetStr := r.URL.Query().Get("offset")
	query := r.URL.Query().Get("query")
	orderField := r.URL.Query().Get("order_field")
	orderByStr := r.URL.Query().Get("order_by")
	var orderBy int

	// limit, offset := 0, 0
	var (
		limit  = 0
		offset = 0
	)
	//проверяем интовые значения
	if limitStr != "" {
		if limit, err = strconv.Atoi(limitStr); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Limit must be integer"))
			return
		}
	}
	if offsetStr != "" {
		if offset, err = strconv.Atoi(offsetStr); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Offset must be integer"))
			return
		}
	}
	if orderByStr != "" {
		fmt.Println("order by str ", orderByStr)
		orderBy, err = strconv.Atoi(orderByStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("order_by must be -1, 0 or 1"))
			return
		}
		if orderBy != -1 && orderBy != 0 && orderBy != 1 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("order_by must be -1, 0 or 1"))
			return
		}
	} else {
		orderBy = 0
	}

	//проверка валидности поля orderField
	//работает по полям `Id`, `Age`, `Name`
	//дабы не путаться с регистрами сделаю все в нижнем
	orderField = strings.ToLower(orderField)
	x := orderField != "id"
	x1 := orderField != "age"
	x2 := orderField != "name"
	if x && x1 && x2 && orderField != "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("order_field работает по полям `Id`, `Age`, `Name`"))
		return
	}
	//если пустой - то возвращаем по `Name`
	if orderField == "" {
		orderField = "name"
	}

	//начинаем поиск по query
	resultRows := make([]Row, 0, 0)
	if query != "" {
		for _, row := range rows {
			name := row.LastName + " " + row.FirstName
			if strings.Contains(name, query) || strings.Contains(row.About, query) {
				resultRows = append(resultRows, row)
			}
		}
	} else {
		resultRows = rows
	}

	//сортировки
	if orderBy != 0 {
		switch orderField {
		case "name":
			if orderBy == 1 {
				sort.Sort(ByName(rows))
			}
			if orderBy == -1 {
				sort.Sort(sort.Reverse(ByName(rows)))
			}
		case "id":
			if orderBy == 1 {
				sort.Sort(ById(rows))
			}
			if orderBy == -1 {
				sort.Sort(sort.Reverse(ById(rows)))
			}
		case "age":
			if orderBy == 1 {
				sort.Sort(ByAge(rows))
			}
			if orderBy == -1 {
				sort.Sort(sort.Reverse(ByAge(rows)))
			}
		}
	}

	lastElement := offset + limit
	if lastElement > 0 {
		if len(rows) >= lastElement {
			rows = rows[offset:lastElement]
		}
		if len(rows) >= offset {
			rows = rows[offset:]
		}
		if len(rows) < offset {
			rows = make([]Row, 0)
		}
	}
	users := make([]User, 0, len(rows))
	for _, row := range rows {
		var user User
		user.Id = row.Id
		user.Name = row.LastName + " " + row.FirstName
		user.About = row.About
		user.Age = row.Age
		user.Gender = row.Gender
		users = append(users, user)
	}

	jsonResult, err := json.Marshal(users)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonResult)
}

type Root struct {
	XMLName xml.Name `xml:"root"`
	Rows    []Row    `xml:"row"`
}

type Row struct {
	Id            int    `xml:"id"`
	Guid          string `xml:"guid"`
	IsActive      string `xml:"isActive"`
	Balance       string `xml:"balance"`
	Picture       string `xml:"picture"`
	Age           int    `xml:"age"`
	EyeColor      string `xml:"eyeColor"`
	FirstName     string `xml:"first_name"`
	LastName      string `xml:"last_name"`
	Gender        string `xml:"gender"`
	Company       string `xml:"company"`
	Email         string `xml:"email"`
	Phone         string `xml:"phone"`
	Address       string `xml:"address"`
	About         string `xml:"about"`
	Registered    string `xml:"registered"`
	FavoriteFruit string `xml:"favoriteFruit"`
}

type ByAge []Row

func (a ByAge) Len() int           { return len(a) }
func (a ByAge) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByAge) Less(i, j int) bool { return a[i].Age < a[j].Age }

type ById []Row

func (a ById) Len() int           { return len(a) }
func (a ById) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ById) Less(i, j int) bool { return a[i].Id < a[j].Id }

type ByName []Row

func (a ByName) Len() int      { return len(a) }
func (a ByName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByName) Less(i, j int) bool {
	return strings.Compare(a[i].LastName+" "+a[i].FirstName, a[j].LastName+" "+a[j].FirstName) < 0
}

type Test struct {
}

func TestClientAllOk(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))

	sr := SearchRequest{
		Limit:      2,
		Offset:     0,
		Query:      "",
		OrderField: "",
		OrderBy:    0,
	}
	sc := &SearchClient{
		AccessToken: "TestToken",
		URL:         ts.URL,
	}

	var sresp *SearchResponse
	var err error
	if sresp, err = sc.FindUsers(sr); err != nil {
		fmt.Println("error happened: ", err)
		return
	}

	fmt.Printf("letter\n")
	fmt.Println(sresp.Users)
}
