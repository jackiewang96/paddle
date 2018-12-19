package main

import (
	"fmt"
	"math/rand"
	"encoding/json"
	"strconv"
	"time"
	"strings"
	"github.com/IsolationWyn/paddle/cgroups"
	"github.com/IsolationWyn/paddle/cgroups/subsystems"
	"github.com/IsolationWyn/paddle/container"
	log "github.com/sirupsen/logrus"
	"os"
)

func Run(tty bool, comArray []string, res *subsystems.ResourceConfig, volume string, containerName string) {
	parent, writePipe := container.NewParentProcess(tty, volume)
	if parent == nil {
		log.Errorf("New parent process error")
		return
	}
	if err := parent.Start(); err != nil {
		log.Error(err)
	}

	// 创建cgroup manager, 并通过调用set和apply设置资源限制并使限制在容器上生效
	cgroupManager := cgroups.NewCgroupManager("paddle")
	defer cgroupManager.Destroy()
	// 设置资源限制
	cgroupManager.Set(res)
	// 将容器进程加入到各个subsystem挂载对应的cgroup中
	cgroupManager.Apply(parent.Process.Pid)
	// 对容器设置完限制之后, 初始化容器

	// mntURL := "/root/mnt"
	// rootURL := "/root/"
	// container.DeleteWorkSpace(rootURL, mntURL, volume)

	// 记录容器信息
	containerName, err := recordContainerInfo(parent.Process.Pid, comArray, containerName)
	if err != nil {
		log.Errorf("Record container info error %v", err)
		return
	}
	sendInitCommand(comArray, writePipe)
	if tty {
		parent.Wait()
		deleteContainerInfo(containerName)
	}
}

func sendInitCommand(comArray []string, writePipe *os.File) {
	command := strings.Join(comArray, " ")
	log.Infof("command all is %s", command)
	writePipe.WriteString(command)
	writePipe.Close()
}

func randStringBytes(n int) string {
	letterBytes := "1234567890"
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func recordContainerInfo(containerPID int, commandArray []string, containerName string) (string, error) {
	// 首先生成10位数字的容器ID
	id := randStringBytes(10)
	createTime := time.Now().Format("2006-01-02 15:04:05")
	command := strings.Join(commandArray, "")
	// 如果没有指定容器名, 那么就叫"深海の女の子" (′゜ω。‵)
	if containerName == "" {
		containerName = "深海の女の子"
	}
	// 生成容器信息的结构体实例
	containerInfo := &container.ContainerInfo {
		Id:				id,
		Pid:			strconv.Itoa(containerPID),
		Command:		command,
		CreatedTime:	createTime,
		Status:			container.RUNNING,
		Name:			containerName,
	}
	
	// 将容器信息的对象json序列化成字符串
	jsonBytes, err := json.Marshal(containerInfo)
	if err != nil {
		log.Errorf("Record container info error %v", err)
		return "", err
	}
	jsonStr := string(jsonBytes)
	
	// 生成容器存储路径
	dirUrl := fmt.Sprintf(container.DefaultInfoLocation, containerName)
	// 如果该路径不存在则级联创建
	if err := os.MkdirAll(dirUrl, 0622); err != nil {
		log.Errorf("Mkdir error %s error %v", dirUrl, err)
		return "", err
	}

	// /var/run/paddle/{{containerName}}//config.json
	fileName := dirUrl + "/" + container.ConfigName
	// 创建配置文件 config.json
	file, err := os.Create(fileName)
	defer file.Close()
	if err != nil {
		log.Errorf("Create file %s error %v", fileName, err)
		return "", err
	}
	// 将json化之后的数据写入到文件中
	if _, err := file.WriteString(jsonStr); err != nil {
		log.Errorf("File write string error %v", err)
		return "", err
	}

	return containerName, nil
}

func deleteContainerInfo(containerId string) {
	// 删除容器信息 
	// /var/run/paddle/{{containerId}}
	dirURL := fmt.Sprintf(container.DefaultInfoLocation, containerId)
	if err := os.RemoveAll(dirURL); err != nil {
		log.Errorf("Remove dir %s error %v", dirURL, err)
	}
}