package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
)

/*** Global variables ***/

var db *sql.DB // This variable stores the database connection object
var tmpl *template.Template // This variable stores the parsed templates

/*** Custom Types ***/

// Custom type for a TODO
type Task struct {
	Id			int
	Description	string
	Done		bool
}

// This function is called before the main function.
func init() {
	// Parse and load all templates before starting the server
	tmpl = template.Must(template.ParseGlob("templates/*.html"))
}

// Creates a new connection to the database and stores it in the db variable
func initDB() {
	var err error

	// Open the database connection
	db, err = sql.Open("mysql", "root:toor@(127.0.0.1:3306)/testdb?parseTime=true") 
	if err != nil { log.Fatal(err) }

	// Check if the connection is successful
	if err = db.Ping(); err != nil { log.Fatal(err) }
}

func main() {
	initDB()
	defer db.Close()

	gRouter := mux.NewRouter()

	gRouter.HandleFunc("/", HomeHandler)

	gRouter.HandleFunc("/tasks", fetchTasks).Methods("GET")
	gRouter.HandleFunc("/tasks", addTask).Methods("POST")
	gRouter.HandleFunc("/tasks/{id}", updateTask).Methods("PUT", "POST")
	gRouter.HandleFunc("/tasks/{id}", deleteTask).Methods("DELETE")

	gRouter.HandleFunc("/newTaskForm", getTaskForm).Methods("GET")

	gRouter.HandleFunc("/taskUpdateForm/{id}", getTaskUpdateForm).Methods("GET")

	
	http.ListenAndServe(":8080", gRouter)
}

/*** Handler functions ***/

// This handler function sends a response to the root URL ("/")
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	err := tmpl.ExecuteTemplate(w, "home.html", nil)
	if err != nil { 
		http.Error(w, "Error while loading templates: " + err.Error(), http.StatusInternalServerError) 
	}
}

// This handler function is used to fetch all the tasks from the Database at "/tasks"
func fetchTasks(w http.ResponseWriter, r *http.Request) {
	todos, _ := getTasks(db)
	tmpl.ExecuteTemplate(w, "todoList", todos)
}

// This handler function is used to render the form to create a new task
func getTaskForm(w http.ResponseWriter, r *http.Request) {
	tmpl.ExecuteTemplate(w, "addTaskForm", nil)
}

// This handler function is used to add a new task to the Database
func addTask(w http.ResponseWriter, r *http.Request) {
	task := r.FormValue("task") // task is the name of the input field that we want to get the value from
	query := "INSERT INTO tasks (task) VALUES (?)"
	
	stmt, err := db.Prepare(query)
	if err != nil { log.Fatal(err) }

	defer stmt.Close()

	_, executeErr := stmt.Exec(task)
	if executeErr != nil { log.Fatal(executeErr) }

	todos, _ := getTasks(db)
	tmpl.ExecuteTemplate(w, "todoList", todos)
}

// This handler function is used to render the form to update a task
func getTaskUpdateForm(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	taskId, _ := strconv.Atoi(vars["id"])
	task, err := getTaskById(db, taskId)
	if err != nil { http.Error(w, err.Error(), http.StatusInternalServerError) }

	tmpl.ExecuteTemplate(w, "updateTaskForm", task)
}

// This handler function is used to update a task in the Database. Reloads the updated list of tasks if rows were affected.
func updateTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r) // Get the URL variables
	taskItem := r.FormValue("task") // task is the name of the input field that we want to get the value from
	isDone := r.FormValue("done") // done is the name of the input field that we want to get the value from
	
	var taskStatus bool

	switch strings.ToLower(isDone) {
		case "yes", "on":
			taskStatus = true
		case "no", "off":
			taskStatus = false
		default:
			taskStatus = false
	}

	taskId, _ := strconv.Atoi(vars["id"])
	task := Task{taskId, taskItem, taskStatus}

	query := "UPDATE tasks SET task = ?, done = ? WHERE id = ?"
	result, err := db.Exec(query, task.Description, task.Done, task.Id)
	if err != nil { log.Fatal(err) }

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, fmt.Sprintf("Task with ID %d not found", taskId), http.StatusNotFound)
		return
	}

	// Return the fresh list of tasks if rows were affected
	todos, _ := getTasks(db)
	tmpl.ExecuteTemplate(w, "todoList", todos)
}

// This handler function is used to delete a task from the Database. Reloads the updated list of tasks if rows were affected.
func deleteTask(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r) // Get the URL variables
	taskId, _ := strconv.Atoi(vars["id"])

	query := "DELETE FROM tasks WHERE id = ?"
	stmt, err := db.Prepare(query)
	if err != nil { log.Fatal(err) }
	defer stmt.Close()

	_, deleteErr := db.Exec(query, taskId)
	if deleteErr != nil { log.Fatal(deleteErr) }

	// Return the fresh list of tasks if no errors occurred
	todos, _ := getTasks(db)
	tmpl.ExecuteTemplate(w, "todoList", todos)
}

/*** Utility functions ***/

// This utility function makes a query to the DB to get all data saved in the "tasks" table: (SELECT * FROM tasks)
func getTasks(dbPointer *sql.DB) ([]Task, error) {
	query := "SELECT * FROM tasks"
	rows, err := dbPointer.Query(query)
	if err != nil { return nil, err }
	
	defer rows.Close()
	
	var tasks []Task
	for rows.Next(){
		var todo Task
		rowErr := rows.Scan(&todo.Id, &todo.Description, &todo.Done)
		if rowErr != nil { return nil, rowErr }
		tasks = append(tasks, todo)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

// This utility function makes a query to the DB to get a single task by its ID
func getTaskById(dbPointer *sql.DB, id int) (*Task, error) {
	query := "SELECT * FROM tasks WHERE id = ?"
	row := dbPointer.QueryRow(query, id)

	var todo Task
	err := row.Scan(&todo.Id, &todo.Description, &todo.Done)
	if err != nil { 
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("Task with ID %d not found", id)
		}
		return nil, err 
	}

	return &todo, nil
}