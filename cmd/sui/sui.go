package sui

var data string
var debug bool

func init() {
	WatchCmd.PersistentFlags().StringVarP(&data, "data", "d", "::{}", L("Session Data"))
	BuildCmd.PersistentFlags().StringVarP(&data, "data", "d", "::{}", L("Session Data"))
	BuildCmd.PersistentFlags().BoolVarP(&debug, "debug", "D", false, L("Debug mode"))
}
