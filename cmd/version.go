package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/yaoapp/yao/share"
	"runtime"
	"strings"
)

var printAllVersion bool
var versionTemplate = `Version:	  %s
Go version:	  %s
Git commit:	  %s
Built:	          %s
OS/Arch:	  %s/%s
`
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: L("Show version"),
	Long:  L("Show version"),
	Run: func(cmd *cobra.Command, args []string) {
		if printAllVersion {
			commit := strings.Split(share.PRVERSION, "-")[0]
			buildTime := strings.TrimPrefix(share.PRVERSION, commit+"-")
			fmt.Printf(versionTemplate,
				share.VERSION,
				runtime.Version(),
				commit, buildTime,
				runtime.GOOS,
				runtime.GOARCH)
			return
		}
		// Do Stuff Here
		fmt.Println(share.VERSION)
	},
}

func init() {
	versionCmd.PersistentFlags().BoolVarP(&printAllVersion, "all", "", false, L("Print all version information"))
}
