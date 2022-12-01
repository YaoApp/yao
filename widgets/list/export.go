package list

// Export process & api
func Export() error {
	exportProcess()
	return exportAPI()
}
