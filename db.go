/**
 *                                         ,s555SB@@&amp;
 *                                      :9H####@@@@@Xi
 *                                     1@@@@@@@@@@@@@@8
 *                                   ,8@@@@@@@@@B@@@@@@8
 *                                  :B@@@@X3hi8Bs;B@@@@@Ah,
 *             ,8i                  r@@@B:     1S ,M@@@@@@#8;
 *            1AB35.i:               X@@8 .   SGhr ,A@@@@@@@@S
 *            1@h31MX8                18Hhh3i .i3r ,A@@@@@@@@@5
 *            ;@&amp;i,58r5                 rGSS:     :B@@@@@@@@@@A
 *             1#i  . 9i                 hX.  .: .5@@@@@@@@@@@1
 *              sG1,  ,G53s.              9#Xi;hS5 3B@@@@@@@B1
 *               .h8h.,A@@@MXSs,           #@H1:    3ssSSX@1
 *               s ,@@@@@@@@@@@@Xhi,       r#@@X1s9M8    .GA981
 *               ,. rS8H#@@@@@@@@@@#HG51;.  .h31i;9@r    .8@@@@BS;i;
 *                .19AXXXAB@@@@@@@@@@@@@@#MHXG893hrX#XGGXM@@@@@@@@@@MS
 *                s@@MM@@@hsX#@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@&amp;,
 *              :GB@#3G@@Brs ,1GM@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@B,
 *            .hM@@@#@@#MX 51  r;iSGAM@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@8
 *          :3B@@@@@@@@@@@&amp;9@h :Gs   .;sSXH@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@:
 *      s&amp;HA#@@@@@@@@@@@@@@M89A;.8S.       ,r3@@@@@@@@@@@@@@@@@@@@@@@@@@@r
 *   ,13B@@@@@@@@@@@@@@@@@@@5 5B3 ;.         ;@@@@@@@@@@@@@@@@@@@@@@@@@@@i
 *  5#@@#&amp;@@@@@@@@@@@@@@@@@@9  .39:          ;@@@@@@@@@@@@@@@@@@@@@@@@@@@;
 *  9@@@X:MM@@@@@@@@@@@@@@@#;    ;31.         H@@@@@@@@@@@@@@@@@@@@@@@@@@:
 *   SH#@B9.rM@@@@@@@@@@@@@B       :.         3@@@@@@@@@@@@@@@@@@@@@@@@@@5
 *     ,:.   9@@@@@@@@@@@#HB5                 .M@@@@@@@@@@@@@@@@@@@@@@@@@B
 *           ,ssirhSM@&amp;1;i19911i,.             s@@@@@@@@@@@@@@@@@@@@@@@@@@S
 *              ,,,rHAri1h1rh&amp;@#353Sh:          8@@@@@@@@@@@@@@@@@@@@@@@@@#:
 *            .A3hH@#5S553&amp;@@#h   i:i9S          #@@@@@@@@@@@@@@@@@@@@@@@@@A.
 *
 */
package dbrest

import (
	_ "github.com/go-sql-driver/mysql"
)
import (
	"encoding/json"
	"fmt"
	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"
	"github.com/wenlaizhou/middleware"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

var Logger = middleware.GetLogger("dbrest")

type TableHandler struct {
	TableHolder core.Table
	ApiPath     string
	TableName   string
}

type DbApi struct {
	host       string
	port       int
	user       string
	password   string
	db         string
	datasource string
	orm        *xorm.Engine
	dataStruct map[string]reflect.Type
}

var dbApiInstance *DbApi

var dbApiInstanceLock = new(sync.Mutex)

func newXormHandler(host string,
	port int,
	user string,
	password string,
	db string) (*DbApi, error) {
	res := &DbApi{
		host:       host,
		port:       port,
		user:       user,
		password:   password,
		db:         db,
		dataStruct: make(map[string]reflect.Type),
	}
	res.datasource = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8",
		res.user, res.password, res.host, res.port, res.db)
	orm, err := xorm.NewEngine("mysql", res.datasource)
	// orm, err := xorm.NewEngine("sqlite3", "data.db")
	if err != nil {
		Logger.ErrorF("数据库连接错误 %s", err.Error())
		return nil, err
	}
	orm.ShowSQL(true)
	if res.orm != nil {
		_ = res.orm.Close()
	}
	res.orm = orm
	return res, nil
}

func initEngine() {

	dbApiInstanceLock.Lock()
	defer dbApiInstanceLock.Unlock()

	var err error

	// 类型判断
	// var port int
	port, err := middleware.ConfInt(Config, "db.port")
	if err != nil {
		Logger.ErrorF("配置文件端口数据类型错误, 使用默认端口60888: %v", middleware.ConfPrint(Config))
		port = 60888
	}

	dbApiInstance, err = newXormHandler(
		middleware.ConfUnsafe(Config, "db.host"),
		port,
		middleware.ConfUnsafe(Config, "db.user"),
		middleware.ConfUnsafe(Config, "db.password"),
		middleware.ConfUnsafe(Config, "db.database"))

	if middleware.ProcessError(err) {
		return
	}
	dbApiInstance.GetEngine().ShowSQL(true)
}

func (this *DbApi) GetStruct() map[string]map[string]string {
	res := make(map[string]map[string]string)
	for table, st := range this.dataStruct {
		columnStruct := make(map[string]string)
		for i := 0; i < st.NumField(); i++ {
			columnFd := st.Field(i)
			columnStruct[columnFd.Tag.Get("json")] = columnFd.Type.String()
		}
		res[table] = columnStruct
	}
	return res
}

func (this *DbApi) GetMeta(tableName string) core.Table {
	return tableMetas[tableName]
}

func (this *DbApi) GetEngine() *xorm.Engine {
	return this.orm
}

// 获取
func GetMeta(tableName string) core.Table {
	return dbApiInstance.GetMeta(tableName)
}

// 获取数据库引擎
func GetEngine() *xorm.Engine {
	return dbApiInstance.GetEngine()
}

func (this *DbApi) RegisterDbApi(orm interface{}) {
	ormValue := reflect.ValueOf(orm)
	if ormValue.Kind() != reflect.Ptr {
		Logger.Error("orm 对象必须是指针")
		return
	}
	ormType := ormValue.Elem().Type()
	Logger.InfoLn("开始注册 : ", orm.(xorm.TableName).TableName())
	Logger.InfoLn("%#v\n", orm)
	this.dataStruct[orm.(xorm.TableName).TableName()] = ormType
	primaryIndex := -1
	for i := 0; i < ormType.NumField(); i++ {
		tag := ormType.Field(i).Tag.Get("xorm")
		Logger.InfoLn(tag)
		if strings.Contains(tag, "primary") {
			primaryIndex = i
			break
		}
	}
	Logger.InfoLn(primaryIndex)
	isExist, err := this.orm.Exist(orm)
	middleware.ProcessError(err)
	if !isExist {
		_ = this.orm.CreateTables(orm)
	}

	middleware.RegisterHandler(fmt.Sprintf("/%s/insert", orm.(xorm.TableName).TableName()),
		func(ctx middleware.Context) {
			resValue := reflect.New(ormType) // INSERT INTO .. ON DUPLICATE KEY UPDATE
			err := json.Unmarshal(ctx.GetBody(), resValue.Interface())
			if err != nil {
				Logger.InfoLn(err.Error())
				_ = ctx.ApiResponse(-1, "", nil)
				return
			}
			Logger.InfoF("%#v", resValue.Interface())
			_, err = this.orm.Insert(resValue.Interface())
			if err != nil {
				Logger.InfoLn(err.Error())
				_ = ctx.ApiResponse(-1, "", nil)
				return
			}
			_ = ctx.ApiResponse(0, "", nil)
			return
		})

	middleware.RegisterHandler(fmt.Sprintf("/%s/update", orm.(xorm.TableName).TableName()),
		func(ctx middleware.Context) {
			resValue := reflect.New(ormType)
			err := json.Unmarshal(ctx.GetBody(), resValue.Interface())
			if err != nil {
				Logger.InfoLn(err.Error())
				_ = ctx.ApiResponse(-1, "", nil)
				return
			}
			condition := make(map[string]int)
			condition["id"], _ = strconv.Atoi(ctx.Request.URL.Query().Get("id"))
			_, err = this.orm.Update(resValue.Interface(), condition)
			if err != nil {
				Logger.InfoLn(err.Error())
				_ = ctx.ApiResponse(-1, "", nil)
				return
			}
			_ = ctx.ApiResponse(0, "", nil)
			return
		})

	middleware.RegisterHandler(fmt.Sprintf("/%s/delete", orm.(xorm.TableName).TableName()),
		func(ctx middleware.Context) {
			id, _ := strconv.Atoi(ctx.Request.URL.Query().Get("id"))
			_, err := this.orm.Delete(map[string]interface{}{"id": id})
			if err != nil {
				Logger.InfoLn(err.Error())
				_ = ctx.ApiResponse(-1, "", nil)
				return
			}
			_ = ctx.ApiResponse(0, "", nil)
			return
		})

	middleware.RegisterHandler(fmt.Sprintf("/%s/select", orm.(xorm.TableName).TableName()),
		func(ctx middleware.Context) {
			resValue := reflect.New(ormType)
			err := json.Unmarshal(ctx.GetBody(), resValue.Interface())
			if err != nil {
				Logger.InfoLn(err.Error())
				_ = ctx.ApiResponse(-1, "", nil)
				return
			}
			res := reflect.New(reflect.SliceOf(ormType)).Interface()
			err = this.orm.Find(res, resValue.Interface())
			if err != nil {
				Logger.InfoLn(err.Error())
				_ = ctx.ApiResponse(-1, "", nil)
				return
			}
			_ = ctx.ApiResponse(0, "", &res)
			return
		})
}

var reg, _ = regexp.Compile("\\$\\{(.*?)\\}")
var idReg, _ = regexp.Compile("(\\d+)\\.id")

// 写入动作日志
func logSql(request middleware.Context, sql string, values []interface{}) {
	Logger.InfoF("%s, %s\n, %s\n, %#v\n",
		request.RemoteAddr(),
		string(request.Request.UserAgent()), sql, values)
}

// sql 拆解
func explainSql(sql string, ids *[]string) (string, []string) {
	variableNames := reg.FindAllStringSubmatch(sql, -1)
	var variables []string

	for resList := range variableNames {
		resList := resList
		variableNameQute := variableNames[resList][0]
		variableName := variableNames[resList][1]
		switch {
		case variableName == "guid":
			id := middleware.Guid()
			sql = strings.Replace(sql, variableNameQute,
				fmt.Sprintf("\"%s\"", id), 1)
			*ids = append(*ids, id)
			break
		case idReg.MatchString(variableName):
			posStr := idReg.FindAllStringSubmatch(variableName, -1)
			pos, err := strconv.ParseInt(posStr[0][1], 10, 0)
			middleware.ProcessError(err)
			sql = strings.Replace(sql, variableNameQute,
				fmt.Sprintf("\"%s\"", (*ids)[pos]), 1)
			break
		default:
			sql = strings.Replace(sql, variableNameQute, "?", 1)
			variables = append(variables, variableName)
			break
		}

	}
	return sql, variables
}
