package main

import (
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
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

	// limit, offset, orderBy := 0, 0, 0
	var (
		limit   = 0
		offset  = 0
		orderBy = 0
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
		if offset, err = strconv.Atoi(offsetStr); err != nil || offset < 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Offset must be integer more than 0"))
			return
		}
	}
	if orderByStr != "" {
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
		w.Write([]byte(`{"Error":"ErrorBadOrderField"}`))
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
		if len(resultRows) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{}`))
			return
		}
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
		if len(rows) < offset {
			rows = make([]Row, 0)
		}
		if len(rows) >= lastElement {
			rows = rows[offset:lastElement]
		} else if len(rows) >= offset {
			rows = rows[offset:]
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

func (cur *User) Equals(compareTo *User) bool {
	if cur == compareTo {
		return true
	}

	if cur.Id != compareTo.Id || cur.Age != compareTo.Age || cur.About != compareTo.About || cur.Name != compareTo.Name || cur.Gender != compareTo.Gender {
		return false
	}
	return true
}

func TestClientAllOkWithOffset(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	expectedResult := &SearchResponse{
		Users: []User{
			{
				Id:     1,
				Name:   "Mayer Hilda",
				Age:    21,
				About:  "Sit commodo consectetur minim amet ex. Elit aute mollit fugiat labore sint ipsum dolor cupidatat qui reprehenderit. Eu nisi in exercitation culpa sint aliqua nulla nulla proident eu. Nisi reprehenderit anim cupidatat dolor incididunt laboris mollit magna commodo ex. Cupidatat sit id aliqua amet nisi et voluptate voluptate commodo ex eiusmod et nulla velit.\n",
				Gender: "female",
			},
		},
		NextPage: true,
	}

	sr := SearchRequest{
		Limit:      1,
		Offset:     1,
		Query:      "",
		OrderField: "Id",
		OrderBy:    1,
	}

	sc := &SearchClient{
		AccessToken: "TestToken",
		URL:         ts.URL,
	}

	var (
		result *SearchResponse
		err    error
	)

	if result, err = sc.FindUsers(sr); err != nil {
		t.Errorf("error happened: %v", err)
		return
	}

	if len(result.Users) != 1 {
		t.Errorf("test failed - wrong users count")
		return
	}

	if !reflect.DeepEqual(expectedResult, result) {
		t.Errorf("test failed - results not match\nGot:\n%v\nExpected:\n%v", result, expectedResult)
	}
}

func TestClientAllOkLastElements(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	expectedResult := &SearchResponse{
		Users: []User{
			{
				Id:     33,
				Name:   "Snow Twila",
				Age:    36,
				About:  "Sint non sunt adipisicing sit laborum cillum magna nisi exercitation. Dolore officia esse dolore officia ea adipisicing amet ea nostrud elit cupidatat laboris. Proident culpa ullamco aute incididunt aute. Laboris et nulla incididunt consequat pariatur enim dolor incididunt adipisicing enim fugiat tempor ullamco. Amet est ullamco officia consectetur cupidatat non sunt laborum nisi in ex. Quis labore quis ipsum est nisi ex officia reprehenderit ad adipisicing fugiat. Labore fugiat ea dolore exercitation sint duis aliqua.\n",
				Gender: "female",
			},
			{
				Id:     34,
				Name:   "Sharp Kane",
				Age:    34,
				About:  "Lorem proident sint minim anim commodo cillum. Eiusmod velit culpa commodo anim consectetur consectetur sint sint labore. Mollit consequat consectetur magna nulla veniam commodo eu ut et. Ut adipisicing qui ex consectetur officia sint ut fugiat ex velit cupidatat fugiat nisi non. Dolor minim mollit aliquip veniam nostrud. Magna eu aliqua Lorem aliquip.\n",
				Gender: "male",
			},
		},
		NextPage: false,
	}

	sr := SearchRequest{
		Limit:      10,
		Offset:     33,
		Query:      "",
		OrderField: "",
		OrderBy:    0,
	}

	sc := &SearchClient{
		AccessToken: "TestToken",
		URL:         ts.URL,
	}

	var (
		result *SearchResponse
		err    error
	)

	if result, err = sc.FindUsers(sr); err != nil {
		t.Errorf("error happened: %v", err)
		return
	}

	if !reflect.DeepEqual(expectedResult, result) {
		t.Errorf("test failed - results not match\nGot:\n%v\nExpected:\n%v", result, expectedResult)
	}
}

func TestClientAllOk(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	expectedResult := &SearchResponse{
		Users: []User{
			{
				Id:     0,
				Name:   "Wolf Boyd",
				Age:    22,
				About:  "Nulla cillum enim voluptate consequat laborum esse excepteur occaecat commodo nostrud excepteur ut cupidatat. Occaecat minim incididunt ut proident ad sint nostrud ad laborum sint pariatur. Ut nulla commodo dolore officia. Consequat anim eiusmod amet commodo eiusmod deserunt culpa. Ea sit dolore nostrud cillum proident nisi mollit est Lorem pariatur. Lorem aute officia deserunt dolor nisi aliqua consequat nulla nostrud ipsum irure id deserunt dolore. Minim reprehenderit nulla exercitation labore ipsum.\n",
				Gender: "male",
			},
			{
				Id:     1,
				Name:   "Mayer Hilda",
				Age:    21,
				About:  "Sit commodo consectetur minim amet ex. Elit aute mollit fugiat labore sint ipsum dolor cupidatat qui reprehenderit. Eu nisi in exercitation culpa sint aliqua nulla nulla proident eu. Nisi reprehenderit anim cupidatat dolor incididunt laboris mollit magna commodo ex. Cupidatat sit id aliqua amet nisi et voluptate voluptate commodo ex eiusmod et nulla velit.\n",
				Gender: "female",
			},
		},
		NextPage: true,
	}

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

	var (
		result *SearchResponse
		err    error
	)

	if result, err = sc.FindUsers(sr); err != nil {
		t.Errorf("error happened: %v", err)
		return
	}
	if len(result.Users) != 2 {
		t.Errorf("test failed - wrong users count")
	}

	if !reflect.DeepEqual(expectedResult, result) {
		t.Errorf("test failed - results not match\nGot:\n%v\nExpected:\n%v", result, expectedResult)
	}
}

func TestClientBigLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	sr := SearchRequest{
		Limit:      27,
		Offset:     0,
		Query:      "",
		OrderField: "",
		OrderBy:    0,
	}
	sc := &SearchClient{
		AccessToken: "TestToken",
		URL:         ts.URL,
	}
	var (
		result *SearchResponse
		err    error
	)

	if result, err = sc.FindUsers(sr); err != nil {
		t.Errorf("error happened: %v", err)
		return
	}
	if len(result.Users) != 25 {
		t.Errorf("error happened - limit must be 25, but actual limit is %v", len(result.Users))
	}
}

func TestClientWrongLimit(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	request := SearchRequest{
		Limit:      -1,
		Offset:     0,
		Query:      "",
		OrderField: "",
		OrderBy:    0,
	}

	client := SearchClient{
		AccessToken: "TestToken",
		URL:         ts.URL,
	}

	result, err := client.FindUsers(request)

	if err.Error() != "limit must be > 0" {
		t.Errorf("test failed - wrong error")
	}

	if result != nil || err == nil {
		t.Errorf("test failed - error expected")
	}
}
func TestClientWrongOffset(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	request := SearchRequest{
		Limit:      1,
		Offset:     -1,
		Query:      "",
		OrderField: "",
		OrderBy:    0,
	}

	client := SearchClient{
		AccessToken: "TestToken",
		URL:         ts.URL,
	}

	result, err := client.FindUsers(request)

	if err.Error() != "offset must be > 0" {
		t.Errorf("test failed - wrong error")
	}

	if result != nil || err == nil {
		t.Errorf("test failed - error expected")
	}
}

func TestClientTimeOut(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer ts.Close()
	client := SearchClient{
		AccessToken: "TestToken",
		URL:         ts.URL,
	}
	request := SearchRequest{
		Limit:      1,
		Offset:     0,
		Query:      "",
		OrderField: "",
		OrderBy:    0,
	}
	response, err := client.FindUsers(request)

	if response != nil || err.Error() != "timeout for limit=2&offset=0&order_by=0&order_field=&query=" {
		t.Error("test failed - must be timeout error")
	}
}

func TestClientNotAuthorized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()
	client := SearchClient{
		AccessToken: "WrongToken",
		URL:         ts.URL,
	}
	request := SearchRequest{
		Limit:      1,
		Offset:     0,
		Query:      "",
		OrderField: "",
		OrderBy:    0,
	}
	response, err := client.FindUsers(request)
	if response != nil || err.Error() != "Bad AccessToken" {
		t.Error("test failed - must be auth error")
	}
}

func TestClientBadUrl(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()
	client := SearchClient{
		AccessToken: "TestToken",
		URL:         "BadUrl",
	}
	request := SearchRequest{
		Limit:      1,
		Offset:     0,
		Query:      "",
		OrderField: "",
		OrderBy:    0,
	}
	response, err := client.FindUsers(request)
	if response != nil || err.Error() != `unknown error Get "BadUrl?limit=2&offset=0&order_by=0&order_field=&query=": unsupported protocol scheme ""` {
		t.Error("test failed - must be BadUrl error")
	}
}

func TestClientBadRequest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()
	client := SearchClient{
		AccessToken: "TestToken",
		URL:         ts.URL,
	}
	request := SearchRequest{
		Limit:      1,
		Offset:     0,
		Query:      "",
		OrderField: "badfield",
		OrderBy:    0,
	}
	response, err := client.FindUsers(request)

	if response != nil || err.Error() != `OrderFeld badfield invalid` {
		t.Error("test failed - must be Bad Request error")
	}
}

func TestFindUsersStatusInternalServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(""))
	}))
	defer ts.Close()

	client := SearchClient{
		AccessToken: "TestToken",
		URL:         ts.URL,
	}
	request := SearchRequest{
		Limit:      1,
		Offset:     0,
		Query:      "",
		OrderField: "",
		OrderBy:    0,
	}
	response, err := client.FindUsers(request)
	if response != nil || err.Error() != "SearchServer fatal error" {
		t.Error("test failed - must be SearchServer fatal error")
	}
}

func TestFindUsersUnknownError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	defer ts.Close()

	client := SearchClient{
		AccessToken: "TestToken",
		URL:         ts.URL,
	}
	request := SearchRequest{
		Limit:      1,
		Offset:     1,
		Query:      "SomeWrongParameter",
		OrderField: "",
		OrderBy:    0,
	}
	response, err := client.FindUsers(request)

	if response != nil || !strings.Contains(err.Error(), "unknown bad request") {
		t.Error("test failed - must be unknown bad request fatal error")
	}
}

func TestServerWrongServer(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("{bad Json}"))
	}))
	defer ts.Close()

	client := SearchClient{
		AccessToken: "TestToken",
		URL:         ts.URL,
	}
	request := SearchRequest{
		Limit:      1,
		Offset:     1,
		Query:      "",
		OrderField: "",
		OrderBy:    0,
	}
	response, err := client.FindUsers(request)

	if response != nil || !strings.Contains(err.Error(), "cant unpack error json") {
		t.Error("test failed - must be bad request fatal error")
	}
}

func TestServerWrongData(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("{bad Json}"))
	}))
	defer ts.Close()

	client := SearchClient{
		AccessToken: "TestToken",
		URL:         ts.URL,
	}
	request := SearchRequest{
		Limit:      1,
		Offset:     1,
		Query:      "",
		OrderField: "",
		OrderBy:    0,
	}
	response, err := client.FindUsers(request)

	if response != nil || !strings.Contains(err.Error(), "cant unpack result json") {
		t.Error("test failed - must be cant unpack result json error")
	}
}
