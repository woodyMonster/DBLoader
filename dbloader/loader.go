package dbloader

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql" // 引入mysql驅動
	"fmt"
)

// _db 設定共用db連線
var _db *gorm.DB

// dbConf 預設檔名
var dbConf = "database.json"

// DBPools 存取 DB 相關所需設定值使用
type DBPools struct {
	DefaultSetting defaultSetting
}

type defaultSetting struct {
	DefaultUser   string `json:"defaultUser"`
	DefaultPasswd string `json:"defaultPasswd"`
	DefaultHost   string `json:"defaultHost"`
	DefaultDBName string `json:"defaultDBName"`
	Settings 	  map[string]string `json:"sttings"`
}

// Init 用來初始化設定檔 若沒有偵測到設定檔則創建一個設定檔
func Init(settingKey ...string) *gorm.DB {
	if FileExists(dbConf) {
		conf, err := LoadConfig(dbConf)
		if err != nil {
			log.Print(err)
		}
		_db = initDBPool(ParseConfig(conf, settingKey))
	} else {
		if err := InitFile(dbConf); err != nil {
			log.Print(err)
		}
		log.Printf("no config file, init finish")
	}
	return _db
}

// SetDBConfName 設定要讀取的設定檔名稱
func SetDBConfName(name string) {
	dbConf = name
}

// FileExists 驗證檔案是否存在
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// LoadConfig 讀取config json檔
func LoadConfig(name string) (interface{}, error) {
	file, err := os.Open(name)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	defer file.Close()

	b, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var conf interface{}
	if err = json.Unmarshal(b, &conf); err != nil {
		panic(err)
	}

	return conf, err
}

// InitFile 初始化一份json格式的設定檔
func InitFile(fileName string) error {
	f, err := os.Create(fileName)
	if err != nil {
		panic(err.Error())
	} else {
		defaultData := make(map[string]interface{})
		defaultData["defaultDBName"] = ""
		defaultData["defaultUser"] = ""
		defaultData["defaultPasswd"] = ""
		defaultData["defaultAddr"] = ""
		defaultData["setting"] = map[string]map[string]string{
			"cloudSetting": map[string]string{
				"User" : "",
				"Passwd" : "",
				"Host" : "",
				"DBName" : "",
			},
		}
		defaultJSON, err := json.MarshalIndent(defaultData, "", "	")
		if err != nil {
			log.Println("ERROR:", err)
			return err
		}
		if _, err = f.Write([]byte(defaultJSON)); err != nil {
			return err
		}
	}
	defer func() {
		if err := f.Close(); err != nil {
			panic(err)
		}
	}()
	return nil
}

// ParseConfig 解析json格式的設定檔
func ParseConfig(config interface{}, settingIndex []string) DBPools {
	var dbPools DBPools
	if conf, ok := config.(map[string]interface{}); ok {
		dbPools.DefaultSetting = defaultSetting{
			DefaultUser: conf["defaultUser"].(string),
			DefaultPasswd: conf["defaultPasswd"].(string),
			DefaultHost: conf["defaultHost"].(string),
			DefaultDBName: conf["defaultDBName"].(string),
			Settings: make(map[string]string),
		}
		if len(settingIndex) != 0 {
			if settings, err := conf["setting"].(map[string]interface{}); err {
				if setting, err := settings[settingIndex[0]].(map[string]interface{}); err {
					for settingKey, dbInfo := range setting {
						dbPools.DefaultSetting.Settings[settingKey] = dbInfo.(string)
					}
				} else {
					log.Print("can't find input data.")
				}
			}
		}
	}
	return dbPools
}

// InitDBPool 用來將設定檔存入Database 的 Pool
func initDBPool(dbConf DBPools) *gorm.DB {
	var url string
	url = fmt.Sprintf("%s:%s@(%v)/%s?charset=utf8&parseTime=True&loc=Local",
		dbConf.DefaultSetting.DefaultUser,
		dbConf.DefaultSetting.DefaultPasswd,
		dbConf.DefaultSetting.DefaultHost,
		dbConf.DefaultSetting.DefaultDBName,
	)
	if len(dbConf.DefaultSetting.Settings) != 0 {
		url = fmt.Sprintf("%s:%s@(%v)/%s?charset=utf8&parseTime=True&loc=Local",
			dbConf.DefaultSetting.Settings["User"],
			dbConf.DefaultSetting.Settings["Passwd"],
			dbConf.DefaultSetting.Settings["Host"],
			dbConf.DefaultSetting.Settings["DBName"],
		)
	}

	_db, err := gorm.Open("mysql", url)
	if err != nil {
		panic("connection fail, error=" + err.Error())
	}
	_db.LogMode(true) // show db actions log
	_db.DB().SetMaxOpenConns(100) // 最大連接數
	_db.DB().SetMaxIdleConns(20) // sql空閑連接數, 如果没有sql任務需要執行的連接數大於20，超時的會關閉
	return _db
}
