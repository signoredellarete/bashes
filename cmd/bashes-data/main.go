package main

import (
	"fmt"
	"os"

	"github.com/signoredellarete/bashes/internal/domain"
	"github.com/signoredellarete/bashes/internal/store"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) < 2 {
		return usage()
	}

	switch args[0] {
	case "validate":
		if len(args) != 2 {
			return usage()
		}
		return validate(args[1])
	case "migrate":
		if len(args) != 3 {
			return usage()
		}
		return migrate(args[1], args[2])
	default:
		return usage()
	}
}

func validate(path string) error {
	data, err := store.NewRepository(path).Load()
	if err != nil {
		return err
	}

	fmt.Printf("valid schema version %d\n", data.Version)
	printSummary(data)
	return nil
}

func migrate(inputPath, outputPath string) error {
	data, err := store.NewRepository(inputPath).Load()
	if err != nil {
		return err
	}

	if err := store.NewRepository(outputPath).Save(data); err != nil {
		return err
	}

	fmt.Printf("migrated %s -> %s\n", inputPath, outputPath)
	printSummary(data)
	return nil
}

func printSummary(data domain.Store) {
	subsystems := 0
	for _, host := range data.Hosts {
		subsystems += len(host.Subsystems)
	}
	fmt.Printf("hosts: %d\n", len(data.Hosts))
	fmt.Printf("subsystems: %d\n", subsystems)
}

func usage() error {
	return fmt.Errorf("usage: bashes-data validate <hosts.json> | bashes-data migrate <input.json> <output.json>")
}
