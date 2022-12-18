package dashboard

// Export process & api
func Export() error {
	exportProcess()
	return exportAPI()
}
