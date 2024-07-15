package sui

var data string
var locales string
var debug bool

func init() {
	WatchCmd.PersistentFlags().StringVarP(&data, "data", "d", "::{}", L("Session Data"))
	BuildCmd.PersistentFlags().StringVarP(&data, "data", "d", "::{}", L("Session Data"))
	BuildCmd.PersistentFlags().BoolVarP(&debug, "debug", "D", false, L("Debug mode"))
	TransCmd.PersistentFlags().StringVarP(&data, "data", "d", "::{}", L("Session Data"))
	TransCmd.PersistentFlags().BoolVarP(&debug, "debug", "D", false, L("Debug mode"))
	TransCmd.PersistentFlags().StringVarP(&locales, "locales", "l", "", L("Locales, separated by commas"))
}
