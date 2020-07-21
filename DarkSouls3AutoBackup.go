package main

import (
	"archive/zip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"
)


type Config struct {
	AutoEnable       bool   `json:"auto_enable"`
	AutoTimeInterval int64  `json:"auto_time_interval"`
	BackupPath       string `json:"backup_path"`
	KeepNumber       int    `json:"keep_number"`
	ZipEnable        bool   `json:"zip_enable"`
}

func loadConfig() (*Config, error) {
	jsonFile, err := os.Open("config.json")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("读取配置文件 config.json 成功")
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	var conf Config

	json.Unmarshal(byteValue, &conf)
	return &conf, err
}

func isFolder(dirPath string) (bool, error) {
	info, err := os.Stat(dirPath)
	if err != nil {
		fmt.Println(dirPath + " 文件夹不存在")
		return false, err
	}

	if info.IsDir() {
		return true, nil
	} else {
		fmt.Println(dirPath + " 不是一个文件夹")
		return false, nil
	}
}

func getGameDataPath() string {
	usr, err := user.Current()
	if err != nil {
		fmt.Println("获取系统当前用户失败")
		panic("获取系统当前用户失败")
	}
	userPath := usr.HomeDir
	gamePath := filepath.Join(userPath, "/AppData/Roaming/DarkSoulsIII")
	gamePath = filepath.FromSlash(gamePath)
	fmt.Println("黑暗之魂3的用户数据路径： " + gamePath)
	dirStatus, _ := isFolder(gamePath)
	if !dirStatus {
		panic("黑暗之魂3的用户数据未找到或者不是文件夹，请检查！")
	}
	return gamePath
}

func createBackupPath(backupPath string) {
	dirStatus, _ := isFolder(backupPath)
	if !dirStatus {
		os.MkdirAll(backupPath, 0700)
	}
}

func zipToBackup(backupPath string) error {

	gameDataPath := getGameDataPath()
	if backupPath != "" {
		createBackupPath(backupPath)
	} else {
		backupPath = os.Getenv("AppData")
		backupPath = filepath.Join(backupPath, "DarkSouls3Backup")
		createBackupPath(backupPath)
	}
	nowTime := time.Now()
	zipfileName := nowTime.Format("20060102150405") + ".zip"
	target := filepath.Join(backupPath, zipfileName)

	zipfile, err := os.Create(target)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	defer zipfile.Close()
	archive := zip.NewWriter(zipfile)
	defer archive.Close()

	filepath.Walk(gameDataPath, func(path string, fi os.FileInfo, errBack error) (err error) {
		if errBack != nil {
			return errBack
		}

		// 通过文件信息，创建 zip 的文件信息
		fh, err := zip.FileInfoHeader(fi)
		if err != nil {
			return err
		}

		// 替换文件信息中的文件名
		fh.Name = filepath.Join(nowTime.Format("20060102150405"), strings.TrimPrefix(path, gameDataPath))
		// 这步开始没有加，会发现解压的时候说它不是个目录
		if fi.IsDir() {
			fh.Name += "/"
		}else {
			fh.Method = zip.Deflate
		}

		// 写入文件信息，并返回一个 Write 结构
		w, err := archive.CreateHeader(fh)
		if err != nil {
			return err
		}

		// 检测，如果不是标准文件就只写入头信息，不写入文件数据到 w
		// 如目录，也没有数据需要写
		if !fh.Mode().IsRegular() {
			return nil
		}

		// 打开要压缩的文件
		fr, err := os.Open(path)
		defer fr.Close()
		if err != nil {
			return err
		}

		// 将打开的文件 Copy 到 w
		n, err := io.Copy(w, fr)
		if err != nil {
			return err
		}
		// 输出压缩的内容
		fmt.Printf("成功压缩文件： %s, 共写入了 %d 个字符的数据\n", path, n)
		return err
	})

	fmt.Println("在 " + nowTime.Format("2006年01月02号-15时04分05秒") + " 时候黑暗之魂3存档备份成功")
	return err
}

func copyToBackup(backupPath string) error {
	gameDataPath := getGameDataPath()

	if backupPath != "" {
		createBackupPath(backupPath)
	} else {
		backupPath = os.Getenv("AppData")
		backupPath = filepath.Join(backupPath, "DarkSouls3Backup")
		createBackupPath(backupPath)
	}

	nowTime := time.Now()
	copyDirName := nowTime.Format("20060102150405")
	targetDir := filepath.Join(backupPath, copyDirName)
	createBackupPath(targetDir)

	err := filepath.Walk(gameDataPath, func(path string, f os.FileInfo, err error) error {
		if f == nil {
			return err
		}
		if !f.IsDir() {
			destNewPath := strings.Replace(path, gameDataPath, targetDir, -1)
			destBaseDir := strings.Replace(destNewPath, filepath.Base(destNewPath), "", -1)
			destBaseDir, _ =filepath.Abs(destBaseDir)
			createBackupPath(destBaseDir)
			fmt.Println("复制文件:" + path + " 到 " + destNewPath)
			copyFile(path, destNewPath)
		}
		return nil
	})
	if err != nil {
		fmt.Printf(err.Error())
	}
	return err
}

//生成目录并拷贝文件
func copyFile(src, dest string) (w int64, err error) {
	srcFile, err := os.Open(src)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer srcFile.Close()
	//分割path目录
	destSplitPathDirs := strings.Split(dest, "/")

	//检测时候存在目录
	destSplitPath := ""
	for index, dir := range destSplitPathDirs {
		if index < len(destSplitPathDirs)-1 {
			destSplitPath = destSplitPath + dir + "/"
			b, _ := pathExists(destSplitPath)
			if b == false {
				fmt.Println("创建目录:" + destSplitPath)
				//创建目录
				err := os.Mkdir(destSplitPath, os.ModePerm)
				if err != nil {
					fmt.Println(err)
				}
			}
		}
	}
	dstFile, err := os.Create(dest)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	defer dstFile.Close()

	return io.Copy(dstFile, srcFile)
}

//检测文件夹路径时候存在
func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

var (
	h		bool
	auto	bool
	toZip		bool

	backupPath 		string
	timeInterval	int64
)

func usage() {
	fmt.Fprintf(os.Stderr, `本软件用于黑暗之魂3存档备份
用法: 在CMD中切换到软件所在路径，然后运行 DS3SaveBack.exe [-h] [-auto] [-t time] [-b backupPath]

参数选项:
`)
	flag.PrintDefaults()
}

func init() {
	flag.BoolVar(&h, "h", false, "查看帮助")
	flag.BoolVar(&auto, "auto", false, "开启自动备份")
	flag.BoolVar(&toZip, "zip", false, "开启备份时压缩成zip格式")

	// 注意 `signal`。默认是 -s string，有了 `signal` 之后，变为 -s signal
	flag.StringVar(&backupPath, "b", "", "存档备份路径, 默认为黑暗之魂3数据所在路径")
	flag.Int64Var(&timeInterval, "t", 300, "存档自动备份间隔时间，默认300秒")
	// 改变默认的 Usage
	flag.Usage = usage
}
func main() {
	conf, err := loadConfig()
	if err == nil {
		if conf.AutoEnable {
			if conf.ZipEnable{
				zipToBackup(conf.BackupPath)
			} else {
				copyToBackup(conf.BackupPath)
			}
			time.Sleep(time.Duration(conf.AutoTimeInterval))
		}
		if conf.ZipEnable{
			zipToBackup(conf.BackupPath)
		} else {
			copyToBackup(conf.BackupPath)
		}
		return
	}
	flag.Parse()
	// 没有任何参数
	//if flag.NArg() == 0 {
	//	flag.Usage()
	//	return
	//}
	if h {
		flag.Usage()
		return
	}
	if auto {
		select {
		default:
			if toZip{
				zipToBackup(backupPath)
			} else {
				copyToBackup(backupPath)
			}
			time.Sleep(time.Duration(timeInterval))
		}
	}else {
		if toZip{
			zipToBackup(backupPath)
		} else {
			copyToBackup(backupPath)
		}
	}

}
