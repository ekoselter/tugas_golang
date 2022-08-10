package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"reflect"
	"strconv"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"

	// "github.com/go-playground/validator"
	_ "github.com/go-sql-driver/mysql"
)

var validation = NewValidation()
var TaskModelbaru = NewTaskModel()

type Task struct {
	Id int64
	Task string `validate:"required" label:"Task"`
	Assignee string `validate:"required" label:"Assignee"`
	Deadline string `validate:"required" label:"Deadline"`
	Status string `validate:"required" label:"Status"`
}

type TaskModel struct {
	conn *sql.DB
}

// START validasi inputan

type Validation struct {
	validate *validator.Validate
	trans    ut.Translator
}

func NewValidation() *Validation {
	translator := en.New()
	uni := ut.New(translator, translator)

	trans, _ := uni.GetTranslator("en")

	validate := validator.New()
	en_translations.RegisterDefaultTranslations(validate, trans)

	// register tag label
	validate.RegisterTagNameFunc(func(field reflect.StructField) string {
		name := field.Tag.Get("label")
		return name
	})

	// membuat custom error
	validate.RegisterTranslation("required", trans, func(ut ut.Translator) error {
		return ut.Add("required", "{0} harus diisi", true)
	}, func(ut ut.Translator, fe validator.FieldError) string {
		t, _ := ut.T("required", fe.Field())
		return t
	})

	return &Validation{
		validate: validate,
		trans:    trans,
	}
}

func (v *Validation) Struct(s interface{}) interface{} {
	errors := make(map[string]string)

	err := v.validate.Struct(s)
	if err != nil {
		for _, e := range err.(validator.ValidationErrors) {
			errors[e.StructField()] = e.Translate(v.trans)
		}
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// END validasi inputan

func DBConnection() (*sql.DB, error) {
	dbDriver := "mysql"
	dbUser := "root"
	dbPass := ""
	dbName := "tugas_golang"

	db, err := sql.Open(dbDriver, dbUser+":"+dbPass+"@/"+dbName)
	return db, err
}

func NewTaskModel() *TaskModel {
	conn, err := DBConnection()
	if err != nil {
		panic(err)
	}

	return &TaskModel{
		conn: conn,
	}
}


func (p *TaskModel) FindAll() ([]Task, error) {

	rows, err := p.conn.Query("select * from task")
	if err != nil {
		return []Task{}, err
	}
	defer rows.Close()

	var dataTask []Task
	for rows.Next() {
		var task Task
		rows.Scan(&task.Id,
			&task.Task,
			&task.Assignee,
			&task.Deadline,
			&task.Status)


		dataTask = append(dataTask, task)
	}

	return dataTask, nil

}


func (p *TaskModel) Create(dataInput Task) bool {

	result, err := p.conn.Exec("insert into task (task, assignee, deadline, status) values(?,?,?,?)",
		dataInput.Task, dataInput.Assignee, dataInput.Deadline, dataInput.Status)

	if err != nil {
		fmt.Println(err)
		return false
	}

	lastInsertId, _ := result.LastInsertId()

	return lastInsertId > 0
}

func (p *TaskModel) Find(id int64, dataInput *Task) error {

	return p.conn.QueryRow("select * from task where id = ?", id).Scan(
		&dataInput.Id,
		&dataInput.Task,
		&dataInput.Assignee,
		&dataInput.Deadline,
		&dataInput.Status)
}

func (p *TaskModel) Update(dataInput Task) error {

	_, err := p.conn.Exec(
		"update task set task = ?, assignee = ?, deadline = ? where id = ?",
		dataInput.Task, dataInput.Assignee, dataInput.Deadline, dataInput.Id)

	if err != nil {
		return err
	}

	return nil
}

func (p *TaskModel) Delete(id int64) {
	p.conn.Exec("delete from task where id = ?", id)
}

func (p *TaskModel) Konfirmasi(id int64) {
	p.conn.Exec("update task set status = ? where id = ?","Dikerjakan", id)
}

func Index(response http.ResponseWriter, request *http.Request) {

	task, _ := TaskModelbaru.FindAll()

	dataMasuk := map[string]interface{}{
		"data": task,
	}

	temp, err := template.ParseFiles("views/index.html")
	if err != nil {
		panic(err)
	}
	temp.Execute(response, dataMasuk)
}

func Add(response http.ResponseWriter, request *http.Request) {

	if request.Method == http.MethodGet {
		temp, err := template.ParseFiles("views/add.html")
		if err != nil {
			panic(err)
		}
		temp.Execute(response, nil)
	} else if request.Method == http.MethodPost {

		request.ParseForm()

		var dataInput Task
		dataInput.Task = request.Form.Get("task")
		dataInput.Assignee = request.Form.Get("assignee")
		dataInput.Deadline = request.Form.Get("deadline")
		dataInput.Status = request.Form.Get("status")

		var data = make(map[string]interface{})

		vErrors := validation.Struct(dataInput)

		if vErrors != nil {
			data["dataInput"] = dataInput
			data["validation"] = vErrors
		} else {
			data["pesan"] = "Data berhasil disimpan"
			TaskModelbaru.Create(dataInput)
		}
		// data := map[string]interface{}{
		// 	"pesan": "Data berhasil disimpan",
		// }

		temp, _ := template.ParseFiles("views/add.html")
		temp.Execute(response, data)
	}

}

func Edit(response http.ResponseWriter, request *http.Request) {
	if request.Method == http.MethodGet {

		queryString := request.URL.Query()
		id, _ := strconv.ParseInt(queryString.Get("id"), 10, 64)

		var dataInput Task
		TaskModelbaru.Find(id, &dataInput)

		data := map[string]interface{}{
			"dataInput": dataInput,
		}

		temp, err := template.ParseFiles("views/edit.html")
		if err != nil {
			panic(err)
		}
		temp.Execute(response, data)

	} else if request.Method == http.MethodPost {

		request.ParseForm()

		var dataInput Task
		dataInput.Id, _ = strconv.ParseInt(request.Form.Get("id"), 10, 64)
		dataInput.Task = request.Form.Get("task")
		dataInput.Assignee = request.Form.Get("assignee")
		dataInput.Deadline = request.Form.Get("deadline")
		dataInput.Status = request.Form.Get("status")

		var data = make(map[string]interface{})

		vErrors := validation.Struct(dataInput)

		if vErrors != nil {
			data["dataInput"] = dataInput
			data["validation"] = vErrors
		} else {
			data["pesan"] = "Data berhasil diupdate"
			TaskModelbaru.Update(dataInput)
		}
		// data := map[string]interface{}{
		// 	"pesan": "Data berhasil disimpan",
		// }

		temp, _ := template.ParseFiles("views/edit.html")
		temp.Execute(response, data)
	}

}

func Delete(response http.ResponseWriter, request *http.Request) {
	queryString := request.URL.Query()
	id, _ := strconv.ParseInt(queryString.Get("id"), 10, 64)

	TaskModelbaru.Delete(id)

	http.Redirect(response, request, "/task", http.StatusSeeOther)
}

func Konfirmasi(response http.ResponseWriter, request *http.Request) {
	queryString := request.URL.Query()
	id, _ := strconv.ParseInt(queryString.Get("id"), 10, 64)

	TaskModelbaru.Konfirmasi(id)

	http.Redirect(response, request, "/task", http.StatusSeeOther)
}

func main() {
	http.HandleFunc("/",Index)
	http.HandleFunc("/task", Index)
	http.HandleFunc("/task/index", Index)
	http.HandleFunc("/task/add", Add)
	http.HandleFunc("/task/edit", Edit)
	http.HandleFunc("/task/delete", Delete)
	http.HandleFunc("/task/konfirmasi", Konfirmasi)

	http.ListenAndServe(":3000", nil)
}