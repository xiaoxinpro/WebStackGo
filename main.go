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
	json := make(map[string]string)
	c.BindJSON(&json)
	ret := gin.H{
		"message": "OK",
		"error": 0,
	}
	var ok bool
	switch cmd {
	case "login_path":
		if _, ok = json["path"]; !ok {
			ret["message"] = "无效数据"
			ret["error"] = 100
		} else if len(json["path"]) < 2 {
			ret["message"] = "登陆入口不可为空"
			ret["error"] = 101
		} else if string([]byte(json["path"])[:1]) != "/" {
			ret["message"] = "登陆入口格式错误，必须以/开头。"
			ret["error"] = 102
		} else {
			Login.Path = json["path"]
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
		if IsJsonKey(json, "username") && IsJsonKey(json, "password") && IsJsonKey(json, "password2") {
			json["username"] = strings.TrimSpace(json["username"])
			json["password"] = strings.TrimSpace(json["password"])
			json["password2"] = strings.TrimSpace(json["password2"])
			if len(json["username"]) < 2 {
				ret["key"] = "username"
				ret["message"] = "登陆账号太短，请输入大于2个字符。"
				ret["error"] = 111
			} else if len(json["password"]) < 6 && json["password"] != "" {
				ret["key"] = "password"
				ret["message"] = "登陆密码太短，请输入大于6个字符。"
				ret["error"] = 111
			} else if json["password"] != json["password2"] {
				ret["key"] = "password2"
				ret["message"] = "确认密码与登陆密码不相同，请重新输入。"
				ret["error"] = 111
			} else {
				message := ""
				if Login.Username != json["username"] {
					Login.Username = json["username"]
					message += "登陆账号修改完成，"
				}
				if json["password"] != "" && Login.Password != GetMD5(json["password"]) {
					Login.Password = GetMD5(json["password"])
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
		if IsJsonKey(json, "title") {
			Config.Title = json["title"]
		}
		if IsJsonKey(json, "description") {
			Config.Description = json["description"]
		}
		if IsJsonKey(json, "keywords") {
			Config.Keywords = json["keywords"]
		}
		if IsJsonKey(json, "recordcode") {
			Config.Recordcode = json["recordcode"]
		}
		if IsJsonKey(json, "footer") {
			Config.Footer = json["footer"]
		}
		if IsJsonKey(json, "url") {
			Config.Url = strings.TrimSpace(json["url"])
		}
		if IsJsonKey(json, "port") {
			if port, err := strconv.Atoi(json["port"]); err != nil && port > 0 && port < 65535 {
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
		if IsJsonKey(json, "class1_name") && IsJsonKey(json,"class2_name") {
			classid := GetClassId(json["class1_name"], json["class2_name"])
			fmt.Println(classid, WebStack.Class[classid])
			if IsJsonKey(json,"name") && IsJsonKey(json,"url") && IsJsonKey(json,"mark") && IsJsonKey(json,"img") {
				if AddClassData(classid, json) {
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
 	case "class":
	default:
		c.JSON(http.StatusFound, gin.H{
			"message": "Error 302",
			"error": 302,
		})
	}
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

func AddClassData(classid int, classData map[string]string) bool {
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