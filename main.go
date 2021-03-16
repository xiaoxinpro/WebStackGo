package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type JsLogin struct {
	Username		string			`json:"username"`
	Password		string			`json:"password"`
	Path			string			`json:"path"`
}

type JsConfig struct {
	Title			string			`json:"title"`
	Url				string			`json:"url"`
	Port			int				`json:"port"`
	Keywords		string			`json:"keywords"`
	Description		string			`json:"description"`
	IsFbOpenGraph	bool			`json:"isFbOpenGraph"`
	IsTwitterCards	bool			`json:"isTwitterCards"`
	Recordcode		string			`json:"recordcode"`
	Footer			string			`json:"footer"`
}

type JsMenu struct {
	Menu			string			`json:"menu"`
	Name			string			`json:"name"`
	Icon			string			`json:"icon"`
	Sub				[]JsMenu		`json:"sub"`
	Url				string			`json:"url"`
}

type JsClassItem struct {
	Url				string			`json:"url"`
	Img				string			`json:"img"`
	Name			string			`json:"name"`
	Mark			string			`json:"mark"`
}

type JsClass struct {
	Name			string			`json:"name"`
	Rows			[]JsClassItem	`json:"rows"`
}

type JsWebStack struct {
	Menu			[]JsMenu		`json:"Menu"`
	Class			[]JsClass		`json:"Class"`
}

var (
	Login			JsLogin
	Config			JsConfig
	WebStack		JsWebStack
)

func main() {
	var err error
	err = LoadJsonFile("./json/login.json", &Login)
	if err != nil {
		fmt.Print("加载登陆文件login.json错误：")
		fmt.Println(err)
		return
	}

	err = LoadJsonFile("./json/config.json", &Config)
	if err != nil {
		fmt.Print("加载配置文件config.json错误：")
		fmt.Println(err)
		return
	} else {
		//err = SaveJsonFile("./json/config.json", &Config)
		if Config.Port <= 0 || Config.Port > 65535 {
			Config.Port = 2802
		}
	}

	err = LoadJsonFile("./json/webstack.json", &WebStack)
	if err != nil {
		fmt.Print("加载页面文件webstack.json错误： ")
		fmt.Println(err)
		return
	}

	r:=gin.Default()
	r.Static("/assets", "./public")
	r.LoadHTMLGlob("views/**/*")
	r.GET("/", GetIndex)
	r.GET("/index.html", GetIndex)
	r.GET("/about.html", GetAbout)
	r.GET(Login.Path, GetLogin)
	r.POST(Login.Path, PostLogin)
	r.Use(AuthMiddleWare())
	{
		r.GET("/admin", AuthMiddleWare(), GetAdmin)
		r.POST("/admin", AuthMiddleWare(), PostAdmin)
		r.POST("/admin/upload", AuthMiddleWare(), PostAdminUpload)
	}

	r.Run(fmt.Sprintf("%s:%d",Config.Url, Config.Port))
}

func GetIndex(c *gin.Context)  {
	c.HTML(http.StatusOK, "index/index.html", gin.H{
		"config": Config,
		"webstack": WebStack,
		//"body": template.HTML("<body>I 'm body<body>"),
	})
}

func GetAbout(c *gin.Context)  {
	c.HTML(http.StatusOK, "index/about.html", gin.H{
		"config": Config,
	})
}

func GetAdmin(c *gin.Context)  {
	cmd := c.DefaultQuery("cmd", "null")
	switch cmd {
	case "logout":
		c.SetCookie("webstackgo_token", "", -1, "/", "localhost", false, true)
		c.JSON(http.StatusOK, gin.H{
			"cmd": cmd,
			"message": "退出登陆成功",
			"error": 0,
		})
	case "webstack.json":
		c.JSON(http.StatusOK, WebStack)
	case "menu.json":
		c.JSON(http.StatusOK,WebStack.Menu)
	case "class.json":
		c.JSON(http.StatusOK, WebStack.Class)
	default:
		c.HTML(http.StatusOK, "admin/index.html", gin.H{
			"login": Login,
			"config": Config,
			"webstack": WebStack,
		})
	}
}

func PostAdmin(c *gin.Context) {
	cmd := c.DefaultQuery("cmd", "null")
	jsonMap := make(map[string]string)
	c.BindJSON(&jsonMap)
	ret := gin.H{
		"message": "OK",
		"error": 0,
	}
	var ok bool
	switch cmd {
	case "login_path":
		if _, ok = jsonMap["path"]; !ok {
			ret["message"] = "无效数据"
			ret["error"] = 100
		} else if len(jsonMap["path"]) < 2 {
			ret["message"] = "登陆入口不可为空"
			ret["error"] = 101
		} else if string([]byte(jsonMap["path"])[:1]) != "/" {
			ret["message"] = "登陆入口格式错误，必须以/开头。"
			ret["error"] = 102
		} else {
			Login.Path = jsonMap["path"]
			err := SaveJsonFile("./json/login.json", &Login)
			if err == nil {
				ret["message"] = "登陆入口修改成功，重启WebStaskGo服务后生效。"
				ret["error"] = 0
			} else {
				ret["message"] = err.Error()
				ret["error"] = 103
			}
		}
		c.JSON(http.StatusOK, ret)
	case "user":
		if IsJsonKey(jsonMap, "username") && IsJsonKey(jsonMap, "password") && IsJsonKey(jsonMap, "password2") {
			jsonMap["username"] = strings.TrimSpace(jsonMap["username"])
			jsonMap["password"] = strings.TrimSpace(jsonMap["password"])
			jsonMap["password2"] = strings.TrimSpace(jsonMap["password2"])
			if len(jsonMap["username"]) < 2 {
				ret["key"] = "username"
				ret["message"] = "登陆账号太短，请输入大于2个字符。"
				ret["error"] = 111
			} else if len(jsonMap["password"]) < 6 && jsonMap["password"] != "" {
				ret["key"] = "password"
				ret["message"] = "登陆密码太短，请输入大于6个字符。"
				ret["error"] = 111
			} else if jsonMap["password"] != jsonMap["password2"] {
				ret["key"] = "password2"
				ret["message"] = "确认密码与登陆密码不相同，请重新输入。"
				ret["error"] = 111
			} else {
				message := ""
				if Login.Username != jsonMap["username"] {
					Login.Username = jsonMap["username"]
					message += "登陆账号修改完成，"
				}
				if jsonMap["password"] != "" && Login.Password != GetMD5(jsonMap["password"]) {
					Login.Password = GetMD5(jsonMap["password"])
					message += "登陆密码修改完成，"
				}
				err := SaveJsonFile("./json/login.json", &Login)
				if err == nil {
					ret["message"] = message + "请重新前往登陆页面登陆。"
					ret["error"] = 0
				} else {
					ret["message"] = err.Error()
					ret["error"] = 112
				}
			}
		} else {
			ret["message"] = "缺少有效数据"
			ret["error"] = 110
		}
		c.JSON(http.StatusOK, ret)
	case "stack":
		if IsJsonKey(jsonMap, "title") {
			Config.Title = jsonMap["title"]
		}
		if IsJsonKey(jsonMap, "description") {
			Config.Description = jsonMap["description"]
		}
		if IsJsonKey(jsonMap, "keywords") {
			Config.Keywords = jsonMap["keywords"]
		}
		if IsJsonKey(jsonMap, "recordcode") {
			Config.Recordcode = jsonMap["recordcode"]
		}
		if IsJsonKey(jsonMap, "footer") {
			Config.Footer = jsonMap["footer"]
		}
		if IsJsonKey(jsonMap, "url") {
			Config.Url = strings.TrimSpace(jsonMap["url"])
		}
		if IsJsonKey(jsonMap, "port") {
			if port, err := strconv.Atoi(jsonMap["port"]); err != nil && port > 0 && port < 65535 {
				Config.Port = port
			}
		}
		err := SaveJsonFile("./json/config.json", &Config)
		if err == nil {
			ret["message"] = "网页设置保存完成"
			ret["error"] = 0
		} else {
			ret["message"] = err.Error()
			ret["error"] = 122
		}
		c.JSON(http.StatusOK, ret)
	case "web-add":
		if IsJsonKey(jsonMap, "class1_name") && IsJsonKey(jsonMap,"class2_name") {
			classid := GetClassId(jsonMap["class1_name"], jsonMap["class2_name"])
			fmt.Println(classid, WebStack.Class[classid])
			if IsJsonKey(jsonMap,"name") && IsJsonKey(jsonMap,"url") && IsJsonKey(jsonMap,"mark") && IsJsonKey(jsonMap,"img") {
				if AddWebData(classid, jsonMap) {
					if err := SaveJsonFile("./json/webstack.json", &WebStack); err == nil {
						ret["message"] = "添加网址成功"
						ret["error"] = 0
					} else {
						ret["message"] = err.Error()
						ret["error"] = 133
					}
				} else {
					ret["message"] = "无效的分类名称"
					ret["error"] = 132
				}
			} else {
				ret["message"] = "上报数据不完整"
				ret["error"] = 131
			}
		} else {
			ret["message"] = "上报数据不完整"
			ret["error"] = 130
		}
		c.JSON(http.StatusOK, ret)
	case "web-edit":
		if IsJsonKey(jsonMap, "index") && IsJsonKey(jsonMap, "class1_name") && IsJsonKey(jsonMap,"class2_name") {
			classid := GetClassId(jsonMap["class1_name"], jsonMap["class2_name"])
			//fmt.Println(classid, WebStack.Class[classid])
			if IsJsonKey(jsonMap,"name") && IsJsonKey(jsonMap,"url") && IsJsonKey(jsonMap,"mark") && IsJsonKey(jsonMap,"img") {
				if EditWebData(classid, jsonMap) {
					if err := SaveJsonFile("./json/webstack.json", &WebStack); err == nil {
						ret["message"] = "编辑网址成功"
						ret["error"] = 0
					} else {
						ret["message"] = err.Error()
						ret["error"] = 143
					}
				} else {
					ret["message"] = "无效的网址源信息"
					ret["error"] = 142
				}
			} else {
				ret["message"] = "上报数据不完整"
				ret["error"] = 141
			}
		} else {
			ret["message"] = "上报数据不完整"
			ret["error"] = 140
		}
		c.JSON(http.StatusOK, ret)
	case "web-delete":
		isWebDeleteOk := false
		if IsJsonKey(jsonMap, "index") {
			classid, webid := WebIndex2ID(jsonMap["index"])
			if DeleteWebData(classid, webid) {
				isWebDeleteOk = true;
			} else {
				ret["message"] = "无效的网址源信息"
				ret["error"] = 151
			}
		} else if IsJsonKey(jsonMap, "indexArray") {
			var indexArray []string
			err := json.Unmarshal([]byte(jsonMap["indexArray"]), &indexArray)
			if err == nil {
				for i:=0; i< len(indexArray); i++ {
					classid, webid := WebIndex2ID(indexArray[i])
					DeleteWebData(classid, webid)
				}
				isWebDeleteOk = true;
			} else {
				ret["message"] = "批量删除数据结构错误。"
				ret["error"] = 153
			}
		} else {
			ret["message"] = "上报数据不完整"
			ret["error"] = 150
		}
		if isWebDeleteOk == true {
			if err := SaveJsonFile("./json/webstack.json", &WebStack); err == nil {
				ret["message"] = "删除网址成功"
				ret["error"] = 0
			} else {
				ret["message"] = err.Error()
				ret["error"] = 152
			}
		}
		c.JSON(http.StatusOK, ret)
 	case "class-add":
		if IsJsonKey(jsonMap, "name") && IsJsonKey(jsonMap,"icon") && IsJsonKey(jsonMap,"class_up") && IsJsonKey(jsonMap,"class_id") {
			classup, _ := strconv.Atoi(jsonMap["class_up"])
			if CheckClassName(jsonMap["name"], "") == false {
				ret["message"] = "分类名称冲突，请更改分类名称。"
				ret["error"] = 162
			} else if AddClassData(classup, jsonMap["name"], jsonMap["icon"]) {
				if err := SaveJsonFile("./json/webstack.json", &WebStack); err == nil {
					ret["message"] = "添加分类成功"
					ret["error"] = 0
				} else {
					ret["message"] = err.Error()
					ret["error"] = 163
				}
			} else {
				ret["message"] = "上报数据参数错误"
				ret["error"] = 161
			}
		} else {
			ret["message"] = "上报数据不完整"
			ret["error"] = 160
		}
		c.JSON(http.StatusOK, ret)
	case "class-edit":
		if IsJsonKey(jsonMap, "name") && IsJsonKey(jsonMap,"icon") && IsJsonKey(jsonMap,"class_up") && IsJsonKey(jsonMap,"class_id") {
			classup, _ := strconv.Atoi(jsonMap["class_up"])
			if CheckClassName(jsonMap["name"], jsonMap["class_id"]) == false {
				ret["message"] = "分类名称冲突，请更改分类名称。"
				ret["error"] = 172
			} else if EditClassData(jsonMap["class_id"], classup, jsonMap["name"], jsonMap["icon"]) {
				if err := SaveJsonFile("./json/webstack.json", &WebStack); err == nil {
					ret["message"] = "编辑分类成功"
					ret["error"] = 0
				} else {
					ret["message"] = err.Error()
					ret["error"] = 173
				}
			} else {
				ret["message"] = "上报数据参数错误"
				ret["error"] = 171
			}
		} else {
			ret["message"] = "上报数据不完整"
			ret["error"] = 170
		}
		c.JSON(http.StatusOK, ret)
	case "class-delete":
		isClassDeleteOk := false
		if IsJsonKey(jsonMap, "index") {
			if DeleteClassData(jsonMap["index"]) {
				isClassDeleteOk = true;
			} else {
				ret["message"] = "无效的网址源信息"
				ret["error"] = 181
			}
		}
		if isClassDeleteOk == true {
			if err := SaveJsonFile("./json/webstack.json", &WebStack); err == nil {
				ret["message"] = "删除网址成功"
				ret["error"] = 0
			} else {
				ret["message"] = err.Error()
				ret["error"] = 182
			}
		}
		c.JSON(http.StatusOK, ret)
	case "class-sort":
		if IsJsonKey(jsonMap, "sort") {
			if SortClassData(jsonMap["sort"]) {
				if err := SaveJsonFile("./json/webstack.json", &WebStack); err == nil {
					ret["message"] = "保存分类排序成功"
					ret["error"] = 0
				} else {
					ret["message"] = err.Error()
					ret["error"] = 193
				}
			}
			ret["message"] = "保存分类排序"
			ret["error"] = 0
		} else {
			ret["message"] = "上报数据不完整"
			ret["error"] = 190
		}
		c.JSON(http.StatusOK, ret)
	default:
		c.JSON(http.StatusFound, gin.H{
			"message": "Error 302",
			"error": 302,
		})
	}
}

func PostAdminUpload(c *gin.Context) {
	// https://github.com/gin-gonic/examples/blob/master/upload-file/single/main.go
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": err.Error(),
			"error": 801,
		})
		return
	}

	//获取文件后缀
	existing := strings.ToLower(Ext(file.Filename))
	if existing == ""{
		c.JSON(http.StatusOK, gin.H{
			"message": "文件类型错误，无法上传。",
			"error": 802,
		})
		return
	}
	extStrSlice := []string{".jpg",".png","gif"}
	if !ContainArray(existing, extStrSlice) {
		c.JSON(http.StatusOK, gin.H{
			"message": "文件类型错误，请上传图片文件（jpg、png、gif）。",
			"error": 803,
		})
		return
	}

	filepath := "public/images/uploads/"
	//如果没有filepath文件目录就创建一个
	if _, err := os.Stat(filepath); err != nil {
		if !os.IsExist(err) {
			os.MkdirAll(filepath, os.ModePerm)
		}
	}
	path := filepath + file.Filename //路径+文件名上传

	if err := c.SaveUploadedFile(file, path); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": err.Error(),
			"error": 804,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url": "../assets/images/uploads/" + file.Filename,
		"message": "upload file success.",
		"error": 0,
	})
}

func GetLogin(c *gin.Context)  {
	if c.FullPath() == Login.Path {
		c.HTML(http.StatusOK, "admin/login.html", gin.H{
			"config": Config,
			"success": false,
			"message": "",
		})
	} else {
		c.HTML(http.StatusUnauthorized, "admin/login.html", gin.H{
			"error": 401,
			"message": "The login page has been modified.",
		})
	}
}

func PostLogin(c *gin.Context)  {
	username := strings.TrimSpace(c.DefaultPostForm("username", ""))
	password := GetMD5(strings.TrimSpace(c.DefaultPostForm("password", "webstackgo")))
	if username == Login.Username && password == Login.Password {
		now := time.Now()
		token := GetToken(username, password, now.Unix())
		fmt.Println(token, now)
		c.SetCookie("webstackgo_token", token, 7200, "/", "localhost", false, true)
		c.HTML(http.StatusOK, "admin/login.html", gin.H{
			"config": Config,
			"success": true,
			"message": "登陆成功！",
		})
	} else {
		c.HTML(http.StatusUnauthorized, "admin/login.html", gin.H{
			"config": Config,
			"success": false,
			"message": "登陆失败：用户名或密码错误。",
		})
	}

}

func AuthMiddleWare() gin.HandlerFunc {
	return func(c *gin.Context) {
		if cookie, err := c.Request.Cookie("webstackgo_token"); err == nil {
			token, _ := url.QueryUnescape(cookie.Value)
			arr := strings.Split(token, "|")
			//fmt.Println(token, arr)
			if len(arr) == 2 {
				if intNow, err2 := strconv.ParseInt(arr[1], 10, 64); err2==nil && token==GetToken(Login.Username, Login.Password, intNow) {
					if time.Now().Unix() - intNow < 3600 {
						token = GetToken(Login.Username, Login.Password, time.Now().Unix())
						c.SetCookie("webstackgo_token", token, 7200, "/", "localhost", false, true)
					}
					c.Next()
					return
				}
			}
		}
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized",
		})
		c.Abort()
		return
	}
}

func IsExist(f string) bool {
	_, err := os.Stat(f)
	return err == nil || os.IsExist(err)
}

func IsJsonKey(m map[string]string, k string) bool {
	_, ret := m[k]
	return ret
}

func GetMD5(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

func GetToken(username, password string, now int64) string {
	return fmt.Sprintf("%s|%d",GetMD5(fmt.Sprintf("%s|%s|%d", username, password, now)), now)
}

func SaveFile(path string, data []byte) error {
	err := ioutil.WriteFile(path, data, os.ModePerm)
	return err
}

func SaveJsonFile(path string, obj interface{}) error {
	content, err := json.Marshal(obj)
	if err == nil {
		err = SaveFile(path, content)
	}
	return err
}

func LoadFile(path string) ([]byte, error) {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	return content, err
}

func LoadJsonFile(path string, obj interface{}) error {
	content, err := LoadFile(path)
	if err != nil {
		fmt.Println(err)
		return err
	}
	err = json.Unmarshal(content, obj)
	if err != nil {
		fmt.Println(err)
	}
	return err
}

func WebIndex2ID(index string) (int,int) {
	arrIndex := strings.Split(index, "-")
	classId := -1
	webId := -1
	if len(arrIndex) == 2 {
		num1 , _ := strconv.Atoi(arrIndex[0])
		webId, _ = strconv.Atoi(arrIndex[1])
		if(num1 >= 0 && num1 < len(WebStack.Menu)) {
			classId = GetClassId(WebStack.Menu[num1].Name, "")
		}
	} else if(len(arrIndex) == 3) {
		num1 , _ := strconv.Atoi(arrIndex[0])
		num2 , _ := strconv.Atoi(arrIndex[1])
		webId, _ = strconv.Atoi(arrIndex[2])
		if(num1 >= 0 && num1 < len(WebStack.Menu)) {
			classId = GetClassId(WebStack.Menu[num1].Name, WebStack.Menu[num1].Sub[num2].Name)
		}
	}
	return classId, webId
}

func ClassIndex2ID(index string) (int,int) {
	arrIndex := strings.Split(index, "-")
	classUp := -1
	classId := -1
	if len(arrIndex) == 2 {
		classUp, _ = strconv.Atoi(arrIndex[0])
		classId, _ = strconv.Atoi(arrIndex[1])
	} else if len(arrIndex) == 1 {
		classId, _ = strconv.Atoi(arrIndex[0])
	}
	return classUp, classId
}

func CheckClassName(name string, index string) bool {
	if len(strings.TrimSpace(name)) == 0 {
		return false
	}
	classup,classid := -1,-1
	if index != "" {
		classup,classid = ClassIndex2ID(index)
	}
	for id, menu := range WebStack.Menu {
		if menu.Name == name {
			if(-1 == classup && id == classid) {
				continue
			}
			return false
		} else {
			for subId, subMenu := range menu.Sub {
				if subMenu.Name == name {
					if(id == classup && subId == classid) {
						continue
					}
					return false
				}
			}
		}
	}
	return true
}

func GetClassIndex(name string) int {
	for index, item := range WebStack.Class {
		if item.Name == name {
			return index
		}
	}
	return -1
}

func GetClassId(name1 string, name2 string) int {
	index := 0
	name := ""
	for _, menu := range WebStack.Menu {
		if menu.Name == name1 {
			if len(menu.Sub) > 0 {
				for _, subMenu := range menu.Sub {
					if subMenu.Name == name2 {
						name = name2
					}
				}
			} else {
				name = name1
			}
			break;
		}
		index += len(menu.Sub)
	}
	if name != "" {
		for ; index < len(WebStack.Class); index++ {
			if WebStack.Class[index].Name == name {
				return index
			}
		}
	}
	return -1
}

func AddWebData(classid int, classData map[string]string) bool {
	if classid < 0 || classid > len(WebStack.Class) {
		return false
	}
	WebStack.Class[classid].Rows = append(WebStack.Class[classid].Rows, JsClassItem{
		Url:  classData["url"],
		Img:  classData["img"],
		Name: classData["name"],
		Mark: classData["mark"],
	})
	return true
}

func DeleteWebData(classid int, webid int) bool {
	if classid >= 0 && webid >= 0 && classid < len(WebStack.Class) && webid < len(WebStack.Class[classid].Rows)  {
		WebStack.Class[classid].Rows = append(WebStack.Class[classid].Rows[:webid], WebStack.Class[classid].Rows[webid+1:]...)
		return true
	} else {
		return false
	}
}

func EditWebData(classid int, classData map[string]string) bool {
	oldClassId, oldWebId := WebIndex2ID(classData["index"])
	if oldClassId >= 0 && oldWebId >= 0 {
		if oldClassId == classid {
			WebStack.Class[classid].Rows[oldWebId].Name = classData["name"]
			WebStack.Class[classid].Rows[oldWebId].Url = classData["url"]
			WebStack.Class[classid].Rows[oldWebId].Img = classData["img"]
			WebStack.Class[classid].Rows[oldWebId].Mark = classData["mark"]
			return true
		} else if AddWebData(classid, classData) {
			return DeleteWebData(oldClassId, oldWebId)
		}
	}
	return false
}

func AddClassData(classup int, classname string, classicon string) bool {
	if classup == -1 {
		WebStack.Menu = append(WebStack.Menu, JsMenu{
			Menu: "smooth",
			Name: classname,
			Icon: classicon,
			Sub:  []JsMenu{},
			Url:  "#" + classname,
		})
		return true
	} else if classup >=0 && classup < len(WebStack.Menu) {
		WebStack.Menu[classup].Sub = append(WebStack.Menu[classup].Sub, JsMenu{
			Menu: "smooth",
			Name: classname,
			Icon: classicon,
			Sub:  []JsMenu{},
			Url:  "#" + classname,
		})
		return true
	} else {
		return false
	}

}

func DeleteClassData(classIndex string) bool {
	oldClassUp, oldClassId := ClassIndex2ID(classIndex)
	if oldClassUp == -1 {
		WebStack.Menu = append(WebStack.Menu[:oldClassId], WebStack.Menu[oldClassId+1:]...)
	} else if oldClassUp >= 0 && oldClassUp < len(WebStack.Menu) {
		WebStack.Menu[oldClassUp].Sub = append(WebStack.Menu[oldClassUp].Sub[:oldClassId], WebStack.Menu[oldClassUp].Sub[oldClassId+1:]...)
	} else {
		return false
	}
	return true
}

func EditClassData(classIndex string, classup int, classname string, classicon string) bool {
	oldClassUp, oldClassId := ClassIndex2ID(classIndex)
	if oldClassUp == classup {
		if oldClassUp == -1 {
			WebStack.Menu[oldClassId].Name = classname
			WebStack.Menu[oldClassId].Icon = classicon
		} else if oldClassUp >= 0 && oldClassUp < len(WebStack.Menu) {
			WebStack.Menu[oldClassUp].Sub[oldClassId].Name = classname
			WebStack.Menu[oldClassUp].Sub[oldClassId].Icon = classicon
		} else {
			return false
		}
	} else{
		if AddClassData(classup, classname, classicon) {
			return DeleteClassData(classIndex)
		} else {
			return false
		}
	}
	return true
}

func SortClassData(sortJson string) bool {
	var sortData []interface{}
	json.Unmarshal([]byte(sortJson), &sortData)
	for i:=0; i < len(sortData); i++ {
		if reflect.TypeOf(sortData[i]).Kind() == reflect.String {
			fmt.Println(reflect.ValueOf(sortData[i]).String())
		} else if reflect.TypeOf(sortData[i]).Kind() == reflect.Slice {
			item := reflect.ValueOf(sortData[i])
			for j:=0; j < item.Len(); j++ {
				fmt.Println(reflect.ValueOf(item.Index(j).Interface()).String())
			}
		}

	}
	fmt.Println(sortData)
	return true
}

//Contain 判断obj是否在target中，target支持的类型array,slice,map   false:不在 true:在
func ContainArray(obj interface{}, target interface{}) bool {
	targetValue := reflect.ValueOf(target)
	switch reflect.TypeOf(target).Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < targetValue.Len(); i++ {
			if targetValue.Index(i).Interface() == obj {
				return true
			}
		}
	case reflect.Map:
		if targetValue.MapIndex(reflect.ValueOf(obj)).IsValid() {
			return true
		}
	}

	return false
}

//获取文件的扩展名
func Ext(path string) string {
	for i := len(path) - 1; i >= 0 && path[i] != '/'; i-- {
		if path[i] == '.' {
			return path[i:]
		}
	}
	return ""
}