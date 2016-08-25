package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const (
	errv byte = iota
	old
	new
)

type V struct {
	V string `json:"Version"`
}

func main() {
	var suffix string
	ver, err := getDockerVer()
	if err != nil {
		LogErr(err, "get docker version error")
		os.Exit(1)
	}
	switch ver {
	case old:
		suffix = "_old"
	case new:
		suffix = "_new"
	default:
		LogErr(errors.New("error ver"), "unknow docker version")
		os.Exit(1)
	}
	go func() {
		cmd := exec.Command("/home/work/uploadCadviosrData/cadvisor" + suffix)
		if err := cmd.Start(); err != nil {
			LogErr(err, "start cadvisor fail")
			return
		}

		LogRun("start cadvisor ok")
		cmd.Wait()
		LogErr(errors.New("cadvisor down"), "restart cadvisor")
	}()

	go func() {
		t := time.NewTicker(60 * time.Second)
		for {
			<-t.C
			cmd := exec.Command("/home/work/uploadCadviosrData/uploadCadvisorData" + suffix)
			if err := cmd.Start(); err != nil {
				LogErr(err, "start uploadCadvisorData fail")
				return
			}
			cmd.Wait()
		}
	}()

	for {
		time.Sleep(time.Second * 120)
		if isAlive() {
			clean()
		} else {
			os.Exit(1)
		}
	}

}
func isAlive() bool {
	f, _ := os.OpenFile("test.txt", os.O_CREATE|os.O_APPEND|os.O_RDONLY, 0660)
	defer f.Close()
	read_buf := make([]byte, 32)
	var pos int64 = 0
	n, _ := f.ReadAt(read_buf, pos)
	if n == 0 {
		return false
	}
	return true
}

func clean() {
	f, _ := os.OpenFile("test.txt", os.O_TRUNC, 0660)
	defer f.Close()
}

func getDockerVer() (byte, error) {
	data, err := RequestUnixSocket("/version", "GET")
	var js V
	if err := json.Unmarshal([]byte(data), &js); err != nil {
		return errv, err
	}
	ver := js.V

	subver := strings.Split(ver, ".")
	if subver[0] != "1" {
		return errv, err
	}
	switcher, err := strconv.Atoi(subver[1])
	if err != nil {
		return errv, err
	}
	if switcher > 6 {
		return new, nil
	}
	if switcher <= 6 {
		return old, nil
	}
	return errv, nil
}

func RequestUnixSocket(address, method string) (string, error) {
	DOCKER_UNIX_SOCKET := "unix:///var/run/docker.sock"
	unix_socket_url := DOCKER_UNIX_SOCKET + ":" + address
	u, err := url.Parse(unix_socket_url)
	if err != nil || u.Scheme != "unix" {
		return "", err
	}

	hostPath := strings.Split(u.Path, ":")
	u.Host = hostPath[0]
	u.Path = hostPath[1]

	conn, err := net.Dial("unix", u.Host)
	if err != nil {
		return "", err
	}

	reader := strings.NewReader("")
	query := ""
	if len(u.RawQuery) > 0 {
		query = "?" + u.RawQuery
	}

	request, err := http.NewRequest(method, u.Path+query, reader)
	if err != nil {
		return "", err
	}

	client := httputil.NewClientConn(conn, nil)
	response, err := client.Do(request)
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	defer response.Body.Close()

	return string(body), err
}
