package model

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/schema"
	"github.com/yaoapp/kun/log"
)

// BatchMigrate batch migrate models after checking which tables are missing
// This optimizes the migration process by querying database only once
func BatchMigrate(models map[string]*model.Model) error {
	if len(models) == 0 {
		return nil
	}

	start := time.Now()

	// Get the connector (assume all system/agent models use default connector)
	connector := "default"
	sch := schema.Use(connector)

	// Step 1: Get all existing tables in one query
	existingTables, err := sch.Tables()
	if err != nil {
		return fmt.Errorf("failed to get existing tables: %w", err)
	}

	// Build a map for fast lookup
	tableExists := make(map[string]bool)
	for _, table := range existingTables {
		tableExists[table] = true
	}

	// Step 2: Identify models that need creation (skip existing tables)
	needCreate := make(map[string]*model.Model)

	for id, mod := range models {
		tableName := mod.MetaData.Table.Name
		if tableName == "" {
			log.Warn("Model %s has no table name, skipping", id)
			continue
		}

		if !tableExists[tableName] {
			needCreate[id] = mod
		}
	}

	// Step 3: Create missing tables only
	if len(needCreate) > 0 {
		isDevelopment := os.Getenv("YAO_ENV") == "development"

		if isDevelopment {
			fmt.Printf("  %s Creating %d tables...\n", color.CyanString("→"), len(needCreate))
		}

		for id, mod := range needCreate {
			createStart := time.Now()
			err := mod.CreateTable()
			if err != nil {
				log.Error("Failed to create table for model %s: %s", id, err.Error())
				return fmt.Errorf("failed to create table for %s: %w", id, err)
			}

			duration := time.Since(createStart)
			if isDevelopment {
				fmt.Printf("    %s %s %s\n",
					color.GreenString("✓"),
					mod.MetaData.Table.Name,
					color.GreenString("(%v)", duration))
			} else {
				log.Info("Created table: %s (%v)", mod.MetaData.Table.Name, duration)
			}
		}
	}

	log.Trace("Batch migrate completed: %d models checked, %d tables created (%v)",
		len(models), len(needCreate), time.Since(start))

	return nil
}
