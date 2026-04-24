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

	TestCmd.PersistentFlags().StringVarP(&data, "data", "d", "::{}", L("Session Data"))
	TestCmd.Flags().StringVar(&testPage, "page", "", L("Filter by page route (substring match)"))
	TestCmd.Flags().StringVar(&testRun, "run", "", L("Filter test functions by regex"))
	TestCmd.Flags().BoolVarP(&testVerbose, "verbose", "v", false, L("Verbose output"))
	TestCmd.Flags().BoolVar(&testJSON, "json", false, L("Output report in JSON format"))
	TestCmd.Flags().BoolVar(&testFailFast, "fail-fast", false, L("Stop on first failure"))
	TestCmd.Flags().StringVar(&testTimeout, "timeout", "30s", L("Timeout per test"))
}
