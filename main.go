package main

import (
	"encoding/json"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"strconv"
)

var db *gorm.DB
var DBUser string
var DBPass string
var DBName string

type TodoItemModel struct {
	Id          int `gorm:"primary_key"`
	Description string
	Completed   bool
}

func Init() {
	err := godotenv.Load()

	if err != nil {
		log.Fatalf("Error getting env, not coming through %v", err)
	}

	DBUser = os.Getenv("DB_USER")
	DBPass = os.Getenv("DB_PASSWORD")
	DBName = os.Getenv("DB_NAME")

	db, _ = gorm.Open(
		"mysql",
		DBUser+":"+DBPass+"@/"+DBName+"?charset=utf8&parseTime=True&loc=Local",
	)
}

func Homepage(res http.ResponseWriter, _ *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(res, `{"alive": true}`)
}

func CreateItem(res http.ResponseWriter, req *http.Request) {
	description := req.FormValue("description")
	log.WithFields(log.Fields{"description": description}).Info("Add new todo item. Saving To database.")
	newTodo := &TodoItemModel{Description: description, Completed: false}
	db.Create(&newTodo)
	result := db.Last(&newTodo)
	res.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(res).Encode(result.Value)
}

func GetCompletedItems(res http.ResponseWriter, _ *http.Request) {
	log.Info("Get completed todo items")
	CompletedTodoItems := GetTodoItems(true)
	res.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(res).Encode(CompletedTodoItems)
}

func GetUncompletedItems(res http.ResponseWriter, _ *http.Request) {
	log.Info("Get completed todo items")
	UncompletedTodoItems := GetTodoItems(false)
	res.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(res).Encode(UncompletedTodoItems)
}

func UpdateItem(res http.ResponseWriter, req *http.Request) {
	// Get URL Param
	vars := mux.Vars(req)
	id, _ := strconv.Atoi(vars["id"])

	if IsItemExist(id) == false {
		res.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(res, `{"updated": false, "error": "Record not found"}`)
		return
	}

	completed, _ := strconv.ParseBool(req.FormValue("completed"))
	log.WithFields(log.Fields{"Id": id, "Completed": completed}).Info("Update todo item")
	todo := &TodoItemModel{}
	db.First(&todo, id)
	todo.Completed = completed
	db.Save(&todo)
	res.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(res, `{"updated": true}`)
}

func RemoveItem(res http.ResponseWriter, req *http.Request) {
	// Get URL Param
	vars := mux.Vars(req)
	id, _ := strconv.Atoi(vars["id"])

	if IsItemExist(id) == false {
		res.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(res, `{"deleted": false, "error": "Record not found"}`)
		return
	}

	log.WithFields(log.Fields{"Id": id}).Info("Deleting todo item")
	todo := &TodoItemModel{}
	db.First(&todo, id)
	db.Delete(&todo)
	res.Header().Set("Content-Type", "application/json")
	_, _ = io.WriteString(res, `{"deleted": true}`)
}

func IsItemExist(Id int) bool {
	todo := &TodoItemModel{}
	result := db.First(&todo, Id)

	if result.Error != nil {
		log.Warn("Todo item with id: " + strconv.Itoa(Id) + " not found")
		return false
	}

	return true
}

func GetTodoItems(completed bool) interface{} {
	var todos []TodoItemModel
	TodoItems := db.Where("completed = ?", completed).Find(&todos).Value
	return TodoItems
}

func main() {
	Init()

	defer db.Close()

	//db.Debug().DropTableIfExists(&TodoItemModel{})
	//db.Debug().AutoMigrate(&TodoItemModel{})

	log.Info("Starting Server")
	router := mux.NewRouter()
	router.HandleFunc("/", Homepage).Methods("GET")
	router.HandleFunc("/completed", GetCompletedItems).Methods("GET")
	router.HandleFunc("/uncompleted", GetUncompletedItems).Methods("GET")
	router.HandleFunc("/", CreateItem).Methods("POST")
	router.HandleFunc("/{id}", UpdateItem).Methods("PUT")
	router.HandleFunc("/{id}", RemoveItem).Methods("DELETE")
	_ = http.ListenAndServe(":5000", router)

	handler := cors.New(cors.Options{
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE"},
	}).Handler(router)

	_ = http.ListenAndServe(":5000", handler)
}
