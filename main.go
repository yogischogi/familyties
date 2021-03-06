// Package familyties analyses ancestry information from
// Family Tree's Family Finder matches files.
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/yogischogi/familyties/cousins"
)

func main() {
	var (
		// Command line options
		namescol             = flag.Int("namescol", 12, "Column number for cousin names in CSV file.")
		details              = flag.Bool("details", false, "Performs detailed analysis for locations and surnames.")
		min                  = flag.Int("min", 1, "Prints only locations and names that occur at least <min> times.")
		cluster              = flag.String("cluster", "", "Performs cluster analysis on the cousins who's ancestral surnames or locations match <cluster>.")
		exclude              = flag.String("exclude", "", "Excludes cousins who's ancestral surnames or locations match <exclude>.")
		csvout               = flag.String("csvout", "", "Writes countries and frequencies of cousins to a file in CSV format.")
		unite                = flag.String("unite", "", "Merges input files separated by commas.")
		intersect            = flag.String("intersect", "", "Intersects input files separated by commas.")
		intersectbynalo      = flag.String("intersectbynalo", "", "Intersects input files separated by commas looking for common names and locations.")
		intersectbynames     = flag.String("intersectbynames", "", "Intersects input files separated by commas.")
		intersectbylocations = flag.String("intersectbylocations", "", "Intersects input files separated by commas.")
	)
	flag.Parse()

	var (
		names                map[string]bool
		locations            map[string]bool
		filename             string
		ancestries           cousins.Ancestries
		args                 = flag.Args()
		locationsIntersected = false
		definedCountries     map[string]bool
		err                  error
	)

	// Select between options that are exclusive to each other.
	switch {
	case len(args) > 0:
		filename = args[len(args)-1]
		ancestries, err = cousins.NewAncestries(filename, *namescol-1)
		if err != nil {
			fmt.Printf("Error reading Family Finder matches CSV file %v.\n", err)
			os.Exit(1)
		}
		names = ancestries.Names()
		locations = ancestries.Locations()
	case *unite != "":
		ancestriesList, err := cousins.NewAncestriesList(*unite, *namescol-1)
		if err != nil {
			fmt.Printf("Error reading Family Finder matches CSV file %v.\n", err)
			os.Exit(1)
		}
		fmt.Printf("Uniting files %v.\r\n\r\n", *unite)
		ancestries = ancestriesList.Unite()
		names = ancestries.Names()
		locations = ancestries.Locations()
	case *intersect != "":
		ancestriesList, err := cousins.NewAncestriesList(*intersect, *namescol-1)
		if err != nil {
			fmt.Printf("Error reading Family Finder matches CSV file %v.\n", err)
			os.Exit(1)
		}
		fmt.Printf("Intersecting files %v, looking for identical ancestral information.\r\n\r\n", *intersect)
		ancestries = ancestriesList.Intersect()
		names = ancestriesList.CommonNames()
		locations = ancestriesList.CommonLocations()
		locationsIntersected = true
	case *intersectbynalo != "":
		ancestriesList, err := cousins.NewAncestriesList(*intersectbynalo, *namescol-1)
		if err != nil {
			fmt.Printf("Error reading Family Finder matches CSV file %v.\n", err)
			os.Exit(1)
		}
		fmt.Printf("Intersecting files %v, looking for common names and locations.\r\n\r\n", *intersectbynalo)
		ancestries = ancestriesList.IntersectByNamesAndLocations()
		names = ancestriesList.CommonNames()
		locations = ancestriesList.CommonLocations()
		locationsIntersected = true
	case *intersectbynames != "":
		ancestriesList, err := cousins.NewAncestriesList(*intersectbynames, *namescol-1)
		if err != nil {
			fmt.Printf("Error reading Family Finder matches CSV file %v.\n", err)
			os.Exit(1)
		}
		fmt.Printf("Intersecting files %v, looking for common names.\r\n\r\n", *intersectbynames)
		ancestries = ancestriesList.IntersectByNames()
		names = ancestriesList.CommonNames()
		locations = ancestries.Locations()
	case *intersectbylocations != "":
		ancestriesList, err := cousins.NewAncestriesList(*intersectbylocations, *namescol-1)
		if err != nil {
			fmt.Printf("Error reading Family Finder matches CSV file %v.\n", err)
			os.Exit(1)
		}
		fmt.Printf("Intersecting files %v, looking for common locations.\r\n\r\n", *intersectbylocations)
		ancestries = ancestriesList.IntersectByLocations()
		names = ancestries.Names()
		locations = ancestriesList.CommonLocations()
		locationsIntersected = true
	default:
		fmt.Print("No input filename specified.\r\n")
		os.Exit(1)
	}

	// Remove all elements from PredefinedCountries that were
	// eliminated due to an intersection operation.
	switch locationsIntersected {
	case true:
		definedCountries = make(map[string]bool)
		for country, _ := range PredefinedCountries() {
			if locations[strings.ToLower(country)] {
				definedCountries[country] = true
			}
		}
	case false:
		definedCountries = PredefinedCountries()
	}

	// Exclude
	if *exclude != "" {
		fmt.Printf("Cousins who's ancestral surnames or locations match %v are excluded from analysis.\r\n\r\n", *exclude)
		excludes := strings.Split(*exclude, ",")
		for _, name := range excludes {
			name = strings.TrimSpace(name)
			ancestries = ancestries.Exclude(name)
		}
	}

	// Filter ancestral information for cluster analysis.
	if *cluster != "" {
		fmt.Printf("Cluster analysis for %v.\r\n\r\n", *cluster)
		includes := strings.Split(*cluster, ",")
		var newAncestries cousins.Ancestries
		rest := ancestries
		for _, name := range includes {
			name = strings.TrimSpace(name)
			newElements := rest.Include(name)
			rest = rest.Exclude(name)
			newAncestries = append(newAncestries, newElements...)
		}
		ancestries = newAncestries
	}
	if len(ancestries) == 0 {
		fmt.Print("No data found.\r\n")
		os.Exit(0)
	}

	// Quick analysis for predefined countries.
	countries := ancestries.FrequenciesOf(definedCountries)
	sort.Stable(sort.Reverse(&countries))
	fmt.Print("--- Quick search for predefined countries ---\r\n")
	fmt.Print("Number of cousins:  Ancestry from:\r\n")
	for i := 0; i < countries.Len(); i++ {
		if countries[i].NCousins >= *min {
			fmt.Printf("%v %v\r\n", countries[i].NCousins, countries[i].Name)
		}
	}

	// Write countries and frequencies of cousins to a file in CSV format.
	if *csvout != "" {
		if *csvout == filename {
			fmt.Print("Error, CSV filename identical to file containing family data.\r\n")
		} else {
			// Create a list of heatmap locations.
			// US and USA are substituted by US state names.
			locations := make(map[string]bool)
			for country, _ := range definedCountries {
				locations[country] = true
			}
			for _, state := range usStates {
				locations[state] = true
			}
			delete(locations, "US")
			delete(locations, "USA")

			// Calculate frequencies of cousins.
			regionFreqs := ancestries.FrequenciesOf(locations)
			sort.Stable(sort.Reverse(&regionFreqs))

			// Write result to file.
			err := regionFreqs.WriteCSV(*csvout)
			if err != nil {
				fmt.Printf("Error writing countries to file in CSV format, %v.\r\n", err)
			}
		}
	}

	if *details == false {
		os.Exit(0)
	}

	// Detailed analysis of ancestral locations.
	locFreqs := ancestries.FrequenciesOfLocations(locations)
	sort.Stable(sort.Reverse(&locFreqs))
	fmt.Print("\r\n--- Detailed analysis of ancestral locations ---\r\n")
	fmt.Print("Number of cousins:  Ancestry from:\r\n")
	for i := 0; i < locFreqs.Len(); i++ {
		if locFreqs[i].NCousins >= *min {
			fmt.Printf("%v %v\r\n", locFreqs[i].NCousins, locFreqs[i].Name)
		}
	}

	// Detailed analysis of ancestral surnames.
	nameFreqs := ancestries.FrequenciesOfNames(names)
	sort.Stable(sort.Reverse(&nameFreqs))
	fmt.Print("\r\n--- Detailed analysis of ancestral surnames ---\r\n")
	fmt.Print("Number of cousins:  Ancestral surname:\r\n")
	for i := 0; i < nameFreqs.Len(); i++ {
		if nameFreqs[i].NCousins >= *min {
			fmt.Printf("%v %v\r\n", nameFreqs[i].NCousins, nameFreqs[i].Name)
		}
	}
}
