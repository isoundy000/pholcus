//数据输出
package collector

import (
	"github.com/tealeg/xlsx"
	"gopkg.in/mgo.v2"
	// "gopkg.in/mgo.v2/bson"
	"encoding/csv"
	"github.com/henrylee2cn/pholcus/config"
	"github.com/henrylee2cn/pholcus/reporter"
	"os"
	"strconv"
	"strings"
	// "time"
)

/************************ excel 输出 ***************************/
func (self *Collector) excel(dataIndex int) {
	defer func() {
		if err := recover(); err != nil {
			reporter.Log.Println(err)
		}
	}()

	var file *xlsx.File
	var sheet *xlsx.Sheet
	var row *xlsx.Row
	var cell *xlsx.Cell
	var err error

	folder1 := "data"
	_folder2 := strings.Split(config.StartTime.Format("2006-01-02 15:04:05"), ":")
	folder2 := _folder2[0] + "时" + _folder2[1] + "分" + _folder2[2] + "秒"
	folder2 = folder1 + "/" + folder2
	filename := folder2 + "/" + self.Spider.GetName() + "_" + self.Spider.GetKeyword() + " " + strconv.Itoa(self.sum[0]) + "-" + strconv.Itoa(self.sum[1]) + ".xlsx"

	file = xlsx.NewFile()

	// 添加分类数据工作表
	for Name, Rule := range self.GetRules() {
		// 跳过不输出的数据
		if len(Rule.GetOutFeild()) == 0 {
			continue
		}

		sheet = file.AddSheet(Name)
		row = sheet.AddRow()
		for _, title := range Rule.GetOutFeild() {
			cell = row.AddCell()
			cell.Value = title
		}
		cell = row.AddCell()
		cell.Value = "当前链接"
		cell = row.AddCell()
		cell.Value = "上级链接"
		cell = row.AddCell()
		cell.Value = "下载时间"

		num := 0 //小计
		for _, datacell := range self.DockerQueue.Dockers[dataIndex] {
			if datacell["RuleName"].(string) == Name {
				row = sheet.AddRow()
				for _, title := range Rule.GetOutFeild() {
					cell = row.AddCell()
					cell.Value = datacell["Data"].(map[string]string)[title]
				}
				cell = row.AddCell()
				cell.Value = datacell["Url"].(string)
				cell = row.AddCell()
				cell.Value = datacell["ParentUrl"].(string)
				cell = row.AddCell()
				cell.Value = datacell["DownloadTime"].(string)
				num++
			}
		}

		reporter.Log.Printf("[任务：%v | 关键词：%v | 小类：%v] 输出 %v 条数据！！！\n", self.Spider.GetName(), self.Spider.GetKeyword(), Name, num)

	}

	// 创建/打开目录
	// f1, err := os.Stat(folder1)
	// if err != nil || !f1.IsDir() {
	// 	os.Mkdir(folder1, 0)
	// }

	// f2, err := os.Stat(folder2)
	// if err != nil || !f2.IsDir() {
	// 	os.Mkdir(folder2, 0)
	// }

	f2, err := os.Stat(folder2)
	if err != nil || !f2.IsDir() {
		if err := os.MkdirAll(folder2, 0777); err != nil {
			reporter.Log.Printf("Error: %v\n", err)
		}
	}

	// 保存文件
	err = file.Save(filename)

	if err != nil {
		reporter.Log.Println(err)
	}

}

/************************ CSV 输出 ***************************/
func (self *Collector) csv(dataIndex int) {
	defer func() {
		if err := recover(); err != nil {
			reporter.Log.Println(err)
		}
	}()

	folder1 := "data"
	_folder2 := strings.Split(config.StartTime.Format("2006-01-02 15:04:05"), ":")
	folder2 := _folder2[0] + "时" + _folder2[1] + "分" + _folder2[2] + "秒"
	folder2 = folder1 + "/" + folder2
	filenameBase := folder2 + "/" + self.Spider.GetName() + "_" + self.Spider.GetKeyword() + " " + strconv.Itoa(self.sum[0]) + "-" + strconv.Itoa(self.sum[1])

	// 创建/打开目录
	f2, err := os.Stat(folder2)
	if err != nil || !f2.IsDir() {
		if err := os.MkdirAll(folder2, 0777); err != nil {
			reporter.Log.Printf("Error: %v\n", err)
		}
	}

	// 添加分类数据工作表
	for Name, Rule := range self.GetRules() {
		// 跳过不输出的数据
		if len(Rule.GetOutFeild()) == 0 {
			continue
		}

		file, err := os.Create(filenameBase + " (" + Name + ").csv")

		if err != nil {
			reporter.Log.Println(err)
			continue
		}

		// file.WriteString("\xEF\xBB\xBF") // 写入UTF-8 BOM
		w := csv.NewWriter(file)
		th := Rule.GetOutFeild()
		th = append(th, []string{"当前链接", "上级链接", "下载时间"}...)
		w.Write(th)

		num := 0 //小计
		for _, datacell := range self.DockerQueue.Dockers[dataIndex] {
			if datacell["RuleName"].(string) == Name {
				row := []string{}
				for _, title := range Rule.GetOutFeild() {
					row = append(row, datacell["Data"].(map[string]string)[title])
				}

				row = append(row, datacell["Url"].(string))
				row = append(row, datacell["ParentUrl"].(string))
				row = append(row, datacell["DownloadTime"].(string))
				w.Write(row)

				num++
			}
		}
		// 发送缓存数据流
		w.Flush()
		// 关闭文件
		file.Close()
		// 输出报告
		reporter.Log.Printf("[任务：%v | 关键词：%v | 小类：%v] 输出 %v 条数据！！！\n", self.Spider.GetName(), self.Spider.GetKeyword(), Name, num)
	}
}

/************************ MongoDB 输出 ***************************/

func (self *Collector) mgo(dataIndex int) {
	session, err := mgo.Dial(config.DB_URL) //连接数据库
	if err != nil {
		panic(err)
	}
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)

	db := session.DB(config.DB_NAME)         //数据库名称
	collection := db.C(config.DB_COLLECTION) //如果该集合已经存在的话，则直接返回

	for i, count := 0, len(self.DockerQueue.Dockers[dataIndex]); i < count; i++ {
		err = collection.Insert((interface{})(self.DockerQueue.Dockers[dataIndex][i]))
		if err != nil {
			panic(err)
		}
	}
}
