package cmd

import (
	"backend_crm_piposmart/seeders"
	"os"
)

func RunDBSeedCommand() bool {
	args := os.Args

	if len(args) < 3 {
		return false
	}

	if args[1] != "db:seed" {
		return false
	}

	switch args[2] {

	case "sales":
		return seeders.NewSalesSeeder().Run() == nil

	default:
		return false
	}
}
