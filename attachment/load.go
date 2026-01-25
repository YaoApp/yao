package attachment

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/data"
	"github.com/yaoapp/yao/share"
)

// SystemUploaders system uploaders
var systemUploaders = map[string]string{
	"__yao.attachment": "yao/uploaders/attachment.local.yao",
}

// Load load uploaders
func Load(cfg config.Config) error {
	// Register attachment processes
	Init()

	messages := []string{}

	// Load system uploaders
	err := loadSystemUploaders(cfg)
	if err != nil {
		return err
	}

	// Load filesystem uploaders
	exts := []string{"*.s3.yao", "*.local.yao", "*.s3.json", "*.local.json", "*.s3.jsonc", "*.local.jsonc"}
	err = application.App.Walk("uploaders", func(root, file string, isdir bool) error {
		if isdir {
			return nil
		}

		// Skip if not uploader file
		if !isUploaderFile(file) {
			return nil
		}

		err := loadUploaderFile(root, file, cfg)
		if err != nil {
			messages = append(messages, err.Error())
		}
		return err
	}, exts...)

	if len(messages) > 0 {
		for _, message := range messages {
			log.Error("Load filesystem uploaders error: %s", message)
		}
		return fmt.Errorf("%s", strings.Join(messages, ";\n"))
	}

	return nil
}

// loadSystemUploaders load system uploaders
func loadSystemUploaders(cfg config.Config) error {
	for id, path := range systemUploaders {
		content, err := data.Read(path)
		if err != nil {
			return err
		}

		// Parse uploader config
		var option ManagerOption
		err = application.Parse(path, content, &option)
		if err != nil {
			return err
		}

		// Replace environment variables and paths
		option.ReplaceEnv(cfg.DataRoot)

		// Register the uploader manager
		_, err = Register(id, option.Driver, option)
		if err != nil {
			log.Error("register system uploader %s error: %s", id, err.Error())
			return err
		}

		log.Info("loaded system uploader: %s (%s)", id, option.Label)
	}

	return nil
}

// loadUploaderFile load a single uploader file
func loadUploaderFile(root, file string, cfg config.Config) error {
	// Generate uploader ID
	id := share.ID(root, file)

	// Read file content
	content, err := application.App.Read(file)
	if err != nil {
		return fmt.Errorf("failed to read uploader file %s: %v", file, err)
	}

	// Parse uploader config
	var option ManagerOption
	err = application.Parse(file, content, &option)
	if err != nil {
		return fmt.Errorf("failed to parse uploader file %s: %v", file, err)
	}

	// Validate driver consistency between filename and config
	filenameDriver := extractDriverFromFilename(file)
	if filenameDriver != "" && option.Driver != "" && filenameDriver != option.Driver {
		log.Warn("Driver mismatch in uploader file %s: filename suggests '%s' but config has '%s'",
			file, filenameDriver, option.Driver)
	}

	// Replace environment variables and paths
	option.ReplaceEnv(cfg.DataRoot)

	// Register the uploader manager
	_, err = Register(id, option.Driver, option)
	if err != nil {
		log.Error("register uploader %s error: %s", id, err.Error())
		return fmt.Errorf("failed to register uploader %s: %v", id, err)
	}

	log.Info("loaded uploader: %s (%s)", id, option.Label)
	return nil
}

// isUploaderFile checks if the file is an uploader configuration file
func isUploaderFile(filename string) bool {
	// Accept files with specific driver patterns: *.s3.yao, *.local.yao, etc.
	lower := strings.ToLower(filename)
	return strings.HasSuffix(lower, ".s3.yao") ||
		strings.HasSuffix(lower, ".local.yao") ||
		strings.HasSuffix(lower, ".s3.json") ||
		strings.HasSuffix(lower, ".local.json") ||
		strings.HasSuffix(lower, ".s3.jsonc") ||
		strings.HasSuffix(lower, ".local.jsonc")
}

// extractDriverFromFilename extracts the driver name from filename (e.g., "test.s3.yao" -> "s3")
func extractDriverFromFilename(filename string) string {
	lower := strings.ToLower(filename)

	// Extract driver from patterns like "*.s3.yao", "*.local.json", etc.
	if strings.Contains(lower, ".s3.") {
		return "s3"
	} else if strings.Contains(lower, ".local.") {
		return "local"
	}

	return ""
}
