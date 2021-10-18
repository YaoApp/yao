package main

import (
	"encoding/json"
	"io"
	"os"
	"path"

	"github.com/yaoapp/kun/grpc"
	"github.com/yaoapp/kun/maps"
)

// user Here is a real implementation of plugin that writes to a local file with
// the key name and the contents are the value of the key.
type user struct{ grpc.Plugin }

// Exec 读取
func (user *user) Exec(name string, args ...interface{}) (*grpc.Response, error) {
	user.Logger.Debug("message from user.Exec")
	v := maps.MakeMap()
	v.Set("name", name)
	v.Set("args", args)
	bytes, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return &grpc.Response{Bytes: bytes, Type: "map"}, nil
}

func main() {
	var output io.Writer = os.Stderr
	var logroot = os.Getenv("GOU_TEST_PLG_LOG")
	if logroot != "" {
		logfile, err := os.Create(path.Join(logroot, "user.log"))
		if err == nil {
			output = logfile
		}
	}
	plugin := &user{}
	plugin.SetLogger(output, grpc.Trace)
	grpc.Serve(plugin)
}
