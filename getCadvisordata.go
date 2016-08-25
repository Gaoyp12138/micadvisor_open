package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	cpuNum   float64
	countNum int
)

func pushData() {
	cadvisorData, err := getCadvisorData()
	if err != nil {
		LogErr(err, "getcadvisorData err")
		return
	}

	t := time.Now().Unix()
	timestamp := fmt.Sprintf("%d", t)

	cadvDataForOneContainer := strings.Split(cadvisorData, `"aliases":[`)
	for k := 1; k < len(cadvDataForOneContainer); k++ { //Traversal containers and ignore head

		memLimit := getMemLimit(cadvDataForOneContainer[k]) //cadvisor provide the memlimit

		containerId := getContainerId(cadvDataForOneContainer[k]) //cadvisor provide the containerId

		DockerData, _ := getDockerData(containerId) //get container inspect

		endpoint := getEndPoint(DockerData) //there is the hosts file path in the inpect of container
		//marathonId := getMarathonAppId(DockerData)

		getCpuNum(DockerData) //we need to give the container CPU ENV

		tag := getTag(DockerData) //recode some other message for a container

		ausge, busge := getUsageData(cadvDataForOneContainer[k]) //get 2 usage because some metric recoding Incremental metric

		cpuuage1 := getBetween(ausge, `"cpu":`, `,"diskio":`)
		cpuuage2 := getBetween(busge, `"cpu":`, `,"diskio":`)
		if err := pushCPU(cpuuage1, cpuuage2, timestamp, tag, containerId, endpoint); err != nil { //get cadvisor data about CPU
			LogErr(err, "pushCPU err in pushData")
		}

		diskiouage := getBetween(ausge, `"diskio":`, `,"memory":`)
		if err := pushDiskIo(diskiouage, timestamp, tag, containerId, endpoint); err != nil { //get cadvisor data about DISKIO
			LogErr(err, "pushDiskIo err in pushData")
		}

		memoryuage := getBetween(ausge, `"memory":`, `,"network":`)
		if err := pushMem(memLimit, memoryuage, timestamp, tag, containerId, endpoint); err != nil { //get cadvisor data about Memery
			LogErr(err, "pushMem err in pushData")
		}

		networkuage1 := getBetween(ausge, `"network":`, `,"task_stats":`)
		networkuage2 := getBetween(busge, `"network":`, `,"task_stats":`)
		if err := pushNet(networkuage1, networkuage2, timestamp, tag, containerId, endpoint); err != nil { //get cadvisor data about net
			LogErr(err, "pushNet err in pushData")
		}
	}
}
func getMarathonAppId(str string) string {
	res := regexp.MustCompile(`"MARATHON_APP_ID=(.+?)"`).FindStringSubmatch(str)
	if len(res) > 1 {
		return res[1]
	}
	return ""
}

func pushIt(value, timestamp, metric, tags, containerId, counterType, endpoint string) error {
	var (
		err1 error
		err  error
	)
	err1 = pushItSub(value, timestamp, metric, tags, containerId, counterType, endpoint)
	err = pushItSub(value, timestamp, metric, "", containerId, counterType, endpoint)
	if err1 != nil {
		return err1
	}
	return err
}

func pushItSub(value, timestamp, metric, tags, containerId, counterType, endpoint string) error {
	postThing := `[{"metric": "` + metric + `", "endpoint": "` + endpoint + `", "timestamp": ` + timestamp + `,"step": ` + "60" + `,"value": ` + value + `,"counterType": "` + counterType + `","tags": "` + tags + `"}]`
	LogRun(postThing)
	url := "http://127.0.0.1:1988/v1/push"
	resp, err := http.Post(url,
		"application/x-www-form-urlencoded",
		strings.NewReader(postThing))
	if err != nil {
		LogErr(err, "Post err in pushIt")
		return err
	}
	defer resp.Body.Close()
	_, err1 := ioutil.ReadAll(resp.Body)
	if err1 != nil {
		LogErr(err1, "ReadAll err in pushIt")
		return err1
	}
	return nil
}

func pushCount(metric, usageA, usageB, start, end string, countNum int, timestamp, tags, containerId, endpoint string, weight float64) error {

	temp1, _ := strconv.ParseInt(getBetween(usageA, start, end), 10, 64)
	temp2, _ := strconv.ParseInt(getBetween(usageB, start, end), 10, 64)
	usage := float64(temp2-temp1) / float64(countNum) / weight
	value := fmt.Sprintf("%f", usage)
	if err := pushIt(value, timestamp, metric, tags, containerId, "GAUGE", endpoint); err != nil {
		LogErr(err, "pushIt err in "+metric)
		return err
	}
	return nil
}

func pushNet(networkuage1, networkuage2, timestamp, tags, containerId, endpoint string) error {
	LogRun("pushNet")

	if err := pushCount("net.if.in.bytes", networkuage1, networkuage2, `"rx_bytes":`, `,"rx_packets":`, countNum, timestamp, tags, containerId, endpoint, 1.0); err != nil {
		return err
	}
	if err := pushCount("net.if.in.packets", networkuage1, networkuage2, `"rx_packets":`, `,"rx_errors":`, countNum, timestamp, tags, containerId, endpoint, 1.0); err != nil {
		return err
	}
	if err := pushCount("net.if.in.errors", networkuage1, networkuage2, `"rx_errors":`, `,"rx_dropped":`, countNum, timestamp, tags, containerId, endpoint, 1.0); err != nil {
		return err
	}
	if err := pushCount("net.if.in.dropped", networkuage1, networkuage2, `"rx_dropped":`, `,"tx_bytes":`, countNum, timestamp, tags, containerId, endpoint, 1.0); err != nil {
		return err
	}
	if err := pushCount("net.if.out.bytes", networkuage1, networkuage2, `"tx_bytes":`, `,"tx_packets":`, countNum, timestamp, tags, containerId, endpoint, 1.0); err != nil {
		return err
	}
	if err := pushCount("net.if.out.packets", networkuage1, networkuage2, `"tx_packets":`, `,"tx_errors":`, countNum, timestamp, tags, containerId, endpoint, 1.0); err != nil {
		return err
	}
	if err := pushCount("net.if.out.errors", networkuage1, networkuage2, `"tx_errors":`, `,"tx_dropped":`, countNum, timestamp, tags, containerId, endpoint, 1.0); err != nil {
		return err
	}
	if err := pushCount("net.if.out.dropped", networkuage1, networkuage2, `"tx_dropped":`, `,"tx_bytes":`, countNum, timestamp, tags, containerId, endpoint, 1.0); err != nil {
		return err
	}

	return nil
}

func pushMem(memLimit, memoryusage, timestamp, tags, containerId, endpoint string) error {
	LogRun("pushMem")
	memUsageNum := getBetween(memoryusage, `"usage":`, `,"working_set"`)
	fenzi, _ := strconv.ParseInt(memUsageNum, 10, 64)
	fenmu, err := strconv.ParseInt(memLimit, 10, 64)
	if err == nil {
		memUsage := float64(fenzi) / float64(fenmu)
		if err := pushIt(fmt.Sprint(memUsage), timestamp, "mem.memused.percent", tags, containerId, "GAUGE", endpoint); err != nil {
			LogErr(err, "pushIt err in pushMem")
		}
	}
	if err := pushIt(memUsageNum, timestamp, "mem.memused", tags, containerId, "GAUGE", endpoint); err != nil {
		LogErr(err, "pushIt err in pushMem")
	}

	if err := pushIt(fmt.Sprint(fenmu), timestamp, "mem.memtotal", tags, containerId, "GAUGE", endpoint); err != nil {
		LogErr(err, "pushIt err in pushMem")
	}

	memHotUsageNum := getBetween(memoryusage, `"working_set":`, `,"container_data"`)
	fenzi, _ = strconv.ParseInt(memHotUsageNum, 10, 64)
	memHotUsage := float64(fenzi) / float64(fenmu)
	if err := pushIt(fmt.Sprint(memHotUsage), timestamp, "mem.memused.hot", tags, containerId, "GAUGE", endpoint); err != nil {
		LogErr(err, "pushIt err in pushMem")
	}

	return nil
}

func pushDiskIo(diskiouage, timestamp, tags, containerId, endpoint string) error {
	LogRun("pushDiskIo")
	temp := getBetween(diskiouage, `"io_service_bytes":\[`, `,"io_serviced":`)
	readUsage := getBetween(temp, `,"Read":`, `,"Sync"`)

	if err := pushIt(readUsage, timestamp, "disk.io.read_bytes", tags, containerId, "COUNTER", endpoint); err != nil {
		LogErr(err, "pushIt err in pushDiskIo")
	}

	writeUsage := getBetween(temp, `,"Write":`, `}`)

	if err := pushIt(writeUsage, timestamp, "disk.io.write_bytes", tags, containerId, "COUNTER", endpoint); err != nil {
		LogErr(err, "pushIt err in pushDiskIo")
	}

	return nil
}

func pushCPU(cpuuage1, cpuuage2, timestamp, tags, containerId, endpoint string) error {
	LogRun("pushCPU" + fmt.Sprint(cpuNum))
	if err := pushCount("cpu.busy", cpuuage1, cpuuage2, `{"total":`, `,"per_cpu_usage":`, countNum, timestamp, tags, containerId, endpoint, 10000000*float64(cpuNum)); err != nil {
		return err
	}

	if err := pushCount("cpu.user", cpuuage1, cpuuage2, `"user":`, `,"sy`, countNum, timestamp, tags, containerId, endpoint, 10000000*float64(cpuNum)); err != nil {
		return err
	}

	if err := pushCount("cpu.system", cpuuage1, cpuuage2, `"system":`, `},`, countNum, timestamp, tags, containerId, endpoint, 10000000*float64(cpuNum)); err != nil {
		return err
	}

	percpu1 := strings.Split(getBetween(cpuuage1, `,"per_cpu_usage":\[`, `\],"user":`), `,`)
	percpu2 := strings.Split(getBetween(cpuuage2, `,"per_cpu_usage":\[`, `\],"user":`), `,`)

	metric := fmt.Sprintf("cpu.core.busy")
	for i, _ := range percpu1 {
		temp1, _ := strconv.ParseInt(percpu1[i], 10, 64)
		temp2, _ := strconv.ParseInt(percpu2[i], 10, 64)
		temp3 := temp2 - temp1
		perCpuUsage := fmt.Sprintf("%f", float64(temp3)/10000000)
		if err := pushIt(perCpuUsage, timestamp, metric, tags+",core="+fmt.Sprint(i), containerId, "GAUGE", endpoint); err != nil {
			LogErr(err, "pushIt err in pushCPU")
			return err
		}
	}
	return nil
}

func getCpuNum(dockerdata string) {
	cpuNum = 1.0
	Min := 0.001
	tmp := getBetween(dockerdata, `"CPU=`, `",`)
	if tmp != "" {
		cpuNum, _ = strconv.ParseFloat(tmp, 64)
		if math.Dim(cpuNum, 0.0) < Min {
			cpuNum = 1.0
		}
	}
}

func getTag(DockerData string) string {
	//FIXME:if you need a tag, edit it, get message from dockerData
	tags := ""
	return tags
}

func getMemLimit(str string) string {
	return getBetween(str, `"memory":{"limit":`, `,"`)
}

func getBetween(str, start, end string) string {
	res := regexp.MustCompile(start + `(.+?)` + end).FindStringSubmatch(str)
	if len(res) <= 1 {
		LogErr(errors.New("regexp len < 1"), start+" "+end)
		return ""
	}
	return res[1]
}

func getCadvisorData() (string, error) {
	var (
		resp *http.Response
		err  error
		body []byte
	)
	url := "http://localhost:" + CadvisorPort + "/api/v1.2/docker"
	if resp, err = http.Get(url); err != nil {
		LogErr(err, "Get err in getCadvisorData")
		return "", err
	}
	defer resp.Body.Close()
	if body, err = ioutil.ReadAll(resp.Body); err != nil {
		LogErr(err, "ReadAll err in getCadvisorData")
		return "", err
	}

	return string(body), nil
}

func getUsageData(cadvisorData string) (ausge, busge string) {
	ausge = strings.Split(cadvisorData, `{"timestamp":`)[1]
	if len(strings.Split(cadvisorData, `{"timestamp":`)) < 11 {
		countNum = 1
		busge = strings.Split(cadvisorData, `{"timestamp":`)[2]
	} else {
		busge = strings.Split(cadvisorData, `{"timestamp":`)[11]
		countNum = 10
	}

	return ausge, busge
}

func getContainerId(cadvisorData string) string {

	getContainerId1 := strings.Split(cadvisorData, `],"namespace"`)
	getContainerId2 := strings.Split(getContainerId1[0], `","`)
	getContainerId3 := strings.Split(getContainerId2[1], `"`)
	containerId := getContainerId3[0]

	return containerId
}

func getEndPoint(DockerData string) string {
	endPoint := getBetween(DockerData, `"Endpoint=`, `",`)
	if endPoint != "" {
		return endPoint
	}

	filepath := getBetween(DockerData, `"HostsPath":"`, `",`)
	buf := make(map[int]string, 6)
	inputFile, inputError := os.Open(filepath)
	if inputError != nil {
		LogErr(inputError, "getEndPoint open file err"+filepath)
		return ""
	}
	defer inputFile.Close()

	inputReader := bufio.NewReader(inputFile)
	lineCounter := 0
	for i := 0; i < 2; i++ {
		inputString, readerError := inputReader.ReadString('\n')
		if readerError == io.EOF {
			break
		}
		lineCounter++
		buf[lineCounter] = inputString
	}
	hostname := strings.Split(buf[1], "	")[0]
	return hostname
}

func getDockerData(containerId string) (string, error) {
	str, err := RequestUnixSocket("/containers/"+containerId+"/json", "GET")
	if err != nil {
		LogErr(err, "getDockerData err")
	}
	return str, nil
}

func RequestUnixSocket(address, method string) (string, error) {
	DOCKER_UNIX_SOCKET := "unix:///var/run/docker.sock"
	// Example: unix:///var/run/docker.sock:/images/json?since=1374067924
	unix_socket_url := DOCKER_UNIX_SOCKET + ":" + address
	u, err := url.Parse(unix_socket_url)
	if err != nil || u.Scheme != "unix" {
		LogErr(err, "Error to parse unix socket url "+unix_socket_url)
		return "", err
	}

	hostPath := strings.Split(u.Path, ":")
	u.Host = hostPath[0]
	u.Path = hostPath[1]

	conn, err := net.Dial("unix", u.Host)
	if err != nil {
		LogErr(err, "Error to connect to"+u.Host)
		// fmt.Println("Error to connect to", u.Host, err)
		return "", err
	}

	reader := strings.NewReader("")
	query := ""
	if len(u.RawQuery) > 0 {
		query = "?" + u.RawQuery
	}

	request, err := http.NewRequest(method, u.Path+query, reader)
	if err != nil {
		LogErr(err, "Error to create http request")
		// fmt.Println("Error to create http request", err)
		return "", err
	}

	client := httputil.NewClientConn(conn, nil)
	response, err := client.Do(request)
	if err != nil {
		LogErr(err, "Error to achieve http request over unix socket")
		// fmt.Println("Error to achieve http request over unix socket", err)
		return "", err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		LogErr(err, "Error, get invalid body in answer")
		// fmt.Println("Error, get invalid body in answer")
		return "", err
	}

	defer response.Body.Close()

	return string(body), err
}
