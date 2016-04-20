package main

import (
	"flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
	"github.com/kr/pty"
	"net/http"
	"os"
	"os/exec"
)

var container string

func main() {
	flag.Parse()
	if len(os.Args) < 2 {
		fmt.Printf("Usage: server [containerId]\n")
		return
	}
	container = os.Args[1]
	fmt.Printf("Connecting to container %s\n", container)
	http.HandleFunc("/", reqWs)
	glog.Fatal(http.ListenAndServe(":8009", nil))
}

var wsInit = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func reqWs(resp http.ResponseWriter, request *http.Request) {
	ws, err := wsInit.Upgrade(resp, request, nil)
	if err != nil {
		glog.Fatal(err)
	}

	cmd, err := pty.Start(exec.Command("/usr/local/bin/docker", "exec", "-it", "70c3a89bd9ea", "bash"))
	if err != nil {
		glog.Fatal(err)
	}

	go func() {
		for {
			_, in, err := ws.ReadMessage()
			if err != nil {
				glog.Error(err)
				cmd.Close()
				return
			}
			inLen, err := cmd.Write(in)
			if err != nil {
				glog.Error(err)
				ws.Close()
				return
			}
			if inLen < len(in) {
				panic("pty write overflow")
			}
		}
	}()
	out := make([]byte, wsInit.WriteBufferSize)
	for {
		outLen, err := cmd.Read(out)
		if err != nil {
			glog.Error(err)
			ws.Close()
			return
		}
		err = ws.WriteMessage(websocket.TextMessage, out[:outLen])
		if err != nil {
			glog.Error(err)
			cmd.Close()
			return
		}
	}
}
