// Package familyties analyses ancestry information from
// Family Tree's Family Finder matches files.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"

	"code.google.com/p/familyties/cousins"
)

func main() {
	var (
		namescol = flag.Int("namescol", 9, "Column number for cousin names in CSV file.")
		details  = flag.Bool("details", false, "Performs detailed analysis for locations and surnames.")
		min      = flag.Int("min", 1, "Prints only locations and names that occur at least min times.")
		cluster  = flag.String("cluster", "", "Performs cluster analysis on the cousins who's ancestral surnames or locations match cluster.")
	)
	flag.Parse()
	filename := os.Args[len(os.Args)-1]

	// Read ancestral information from Family Finder matches file.
	ancestries, err := cousins.NewAncestries(filename, *namescol-1)
	if err != nil {
		fmt.Printf("Error reading Family Finder matches CSV file %v.\n", err)
		os.Exit(1)
	}

	// Filter ancestral information for cluster analysis.
	if *cluster != "" {
		fmt.Printf("Cluster analysis for %v.\r\n\r\n", *cluster)
		ancestries = ancestries.Filter(*cluster)
	}
	if len(ancestries) == 0 {
		fmt.Print("No data found.\r\n")
		os.Exit(0)
	}

	// Quick analysis for predefined countries.
	countries := ancestries.FrequenciesOf(predefinedCountries)
	sort.Stable(sort.Reverse(&countries))
	fmt.Print("--- Quick search for predefined countries ---\r\n")
	fmt.Print("Number of cousins:  Ancestry from:\r\n")
	for i := 0; i < countries.Len(); i++ {
		if countries[i].NCousins >= *min {
			fmt.Printf("%v %v\r\n", countries[i].NCousins, countries[i].Name)
		}
	}

	if *details == false {
		os.Exit(0)
	}

	// Detailed analysis of ancestral locations.
	locations := ancestries.FrequenciesOfLocations()
	sort.Stable(sort.Reverse(&locations))
	fmt.Print("\r\n--- Detailed analysis of ancestral locations ---\r\n")
	fmt.Print("Number of cousins:  Ancestry from:\r\n")
	for i := 0; i < locations.Len(); i++ {
		if locations[i].NCousins >= *min {
			fmt.Printf("%v %v\r\n", locations[i].NCousins, locations[i].Name)
		}
	}

	// Detailed analysis of ancestral surnames.
	names := ancestries.FrequenciesOfNames()
	sort.Stable(sort.Reverse(&names))
	fmt.Print("\r\n--- Detailed analysis of ancestral surnames ---\r\n")
	fmt.Print("Number of cousins:  Ancestral surname:\r\n")
	for i := 0; i < names.Len(); i++ {
		if names[i].NCousins >= *min {
			fmt.Printf("%v %v\r\n", names[i].NCousins, names[i].Name)
		}
	}
}
