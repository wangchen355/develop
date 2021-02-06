package main

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"text/template"
	"time"
)

// 留言结构
type Liuyan struct {
	Id      int
	Name    string
	Url     string
	Content string
	Time    int
	Image   string
	ImgPath string
}

func ShowTime(timeUnix int) string {
	t := time.Unix(int64(timeUnix), 0)
	return t.Format("2006-01-02 15:04:05")
}

// 全局变量
var db *sql.DB
var view *template.Template

func main() {
	// 连接数据库
	var err error
	dsn :="root:123456@tcp(127.0.0.1:3306)/mybook"
	db, err =sql.Open("mysql",dsn)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	// 准备模板
	err = LoadTemplate()
	if err != nil {
		panic(err)
	}

	// 注册处理函数
	http.HandleFunc("/load", loadHandler)
	http.HandleFunc("/", listHandler)
	http.HandleFunc("/liuyan", liuyanHandler)
	http.HandleFunc("/del", delHandler)
	http.HandleFunc("/read", uploadHandler)
	http.HandleFunc("/edit", editHandler)

	// 启动服务器
	err = http.ListenAndServe(":12345", nil)
	if err != nil {
		panic(err)
	}
}

// 加载模板
func LoadTemplate() error {
	// 准备模板函数
	funcs := make(template.FuncMap)
	funcs["showtime"] = ShowTime

	// 准备模板
	v := template.New("view")
	v.Funcs(funcs)

	_, err := v.ParseGlob("view/*.html")
	if err != nil {
		return err
	}

	view = v
	return nil
}

// 动态加载模板 /load
func loadHandler(w http.ResponseWriter, req *http.Request) {
	err := LoadTemplate()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	io.WriteString(w, `模板加载完成`)
}

// 显示留言页面 /
func listHandler(w http.ResponseWriter, req *http.Request) {
	// 查询数据
	rows, err := db.Query("select * from liuyan")
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	// 获取数据
	lys := []Liuyan{}
	for rows.Next() {
		ly := Liuyan{}
		err := rows.Scan(&ly.Id, &ly.Name, &ly.Url, &ly.Content, &ly.Time,&ly.Image)
		if nil != err {
			http.Error(w, err.Error(), 500)
			return
		}
		lys = append(lys, ly)
	}


	// 显示数据
	err = view.ExecuteTemplate(w, "index.html", lys)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

// 留言页面 /liuyan
func liuyanHandler(w http.ResponseWriter, req *http.Request) {
	if "POST" == req.Method {
		// 获取参数
		name := strings.TrimSpace(req.FormValue("name"))
		url := strings.TrimSpace(req.FormValue("url"))
		content := strings.TrimSpace(req.FormValue("content"))
		file, handler, err := req.FormFile("image")
		if err != nil{
			fmt.Println(err)
			return
		}
		defer file.Close()
		//上传的文件保存在upload路径下
		ext := path.Ext(handler.Filename)       //获取文件后缀
		fileNewName := string(strconv.Itoa(time.Now().Nanosecond()))+ext

		f, err := os.OpenFile("./upload/"+fileNewName, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil{
			fmt.Println(err)
			return
		}
		defer f.Close()

		io.Copy(f, file)


		// 检查参数
		if name == "" || content == "" {
			io.WriteString(w, "参数错误!\n")
			return
		}

		// sql语句
		sql, err := db.Prepare("insert into liuyan(name, url, content, time,img) values(?, ?, ?, ?,?)")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer sql.Close()

		// sql参数,并执行
		_, err = sql.Exec(name, url, content, time.Now().Unix(),"../upload/"+fileNewName)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		// 跳转
		w.Header().Add("Location", "/")
		w.WriteHeader(302)

		// 提示信息
		io.WriteString(w, "提交成功!\n")

		return
	}

	// 显示表单
	err := view.ExecuteTemplate(w, "liuyan.html", nil)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}
//删除
func delHandler(w http.ResponseWriter, r *http.Request)  {
	r.ParseForm()
	idStr := r.Form.Get("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	sqlStr :="delete from liuyan where id=?"
	ret,err :=db.Exec(sqlStr,id)
	if err!=nil {
		fmt.Printf("delete id failed,err:%v\\n\",err")
	}
	_, err = ret.RowsAffected()
	if err !=nil{
		fmt.Printf("delete id failed,err:%v\n",err)
	}
	// 跳转
	w.Header().Add("Location", "/")
	w.WriteHeader(302)

	// 提示信息
	io.WriteString(w, "删除成功!\n")
}
//显示修改页面
func editHandler(w http.ResponseWriter, r *http.Request){
	r.ParseForm()
	idStr := r.Form.Get("id")
	// 查询数据
	sqlStr :="select * from liuyan where id=?"
	rows, err := db.Query(sqlStr,idStr)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	// 获取数据
	lys := []Liuyan{}
	for rows.Next() {
		ly := Liuyan{}
		err := rows.Scan(&ly.Id, &ly.Name, &ly.Url, &ly.Content, &ly.Time,&ly.Image)
		if nil != err {
			http.Error(w, err.Error(), 500)
			return
		}
		lys = append(lys, ly)
	}

	// 显示数据
	err = view.ExecuteTemplate(w, "update.html", lys)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if "POST" == r.Method {
		// 获取参数
		name := strings.TrimSpace(r.FormValue("name"))
		url := strings.TrimSpace(r.FormValue("url"))
		content := strings.TrimSpace(r.FormValue("content"))
		file, handler, err := r.FormFile("image")
		if err != nil{
			fmt.Println(err)
			return
		}
		defer file.Close()
		//上传的文件保存在upload路径下
		ext := path.Ext(handler.Filename)       //获取文件后缀
		fileNewName := string(strconv.Itoa(time.Now().Nanosecond()))+ext

		f, err := os.OpenFile("./upload/"+fileNewName, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil{
			fmt.Println(err)
			return
		}
		defer f.Close()

		io.Copy(f, file)


		// 检查参数
		if name == "" || content == "" {
			io.WriteString(w, "参数错误!\n")
			return
		}

		// sql语句
		sql, err := db.Prepare("update liuyan  set name=?, url=?, content=?, time=?,img=? where id =?")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		defer sql.Close()

		// sql参数,并执行
		_, err = sql.Exec(name, url, content, time.Now().Unix(),"./upload/"+fileNewName,idStr)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		// 跳转
		w.Header().Add("Location", "/")
		w.WriteHeader(302)
		// 提示信息
		io.WriteString(w, "提交成功!\n")

		return
	}
}
//通过路径显示图片
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	imgStr := r.Form.Get("img")
	imgpath := imgStr
	http.ServeFile(w, r, imgpath)
}