// Package cousins provides classes to work with cousins' ancestral information.
package cousins

import (
	"encoding/csv"
	"errors"
	"os"
	"strings"
)

// tokenDelimiters separate semantical units of text. Each token delimiter is also a word delimiter.
var tokenDelimiters = map[rune]bool{',': true, '/': true, '(': true, ')': true, '-': true, '&': true, ';': true}

// wordDelimiters separate words. Each token delimiter is also a word delimiter.
var wordDelimiters = map[rune]bool{' ': true, '\t': true}

// Ancestry contains a cousin's ancestral surnames and locations.
type Ancestry struct {
	// line is the original line from the Family Finder matches file
	// in small caps.
	line string
	// Words are the words contained in the line.
	Words []string
	// A token consists of one or more words that belong together sematically,
	// for example "United States of America".
	Tokens []string
	// Names are the ancestral surnames.
	Names     []string
	Locations []string
}

// NewAncestry creates an Ancstry from a single line of the
// FamilyFinder matches file. Double entries are eliminated.
// All names and locations are returned in small caps.
func NewAncestry(line string) Ancestry {
	line = strings.ToLower(line)
	names := make(map[string]bool)
	locations := make(map[string]bool)
	tokens := extractTokens(line)
	tokens = normalizeTokens(tokens)
	words := extractWords(line)
	words = normalizeTokens(words)

	// Entries are separated by "/". Each entry contains
	// a name and a sometimes also a location. A location
	// may consist of several parts, for example a town and
	// a country.
	entries := strings.Split(line, "/")
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)

		// Extract name.
		name := ""
		pos := strings.IndexRune(entry, '(')
		if pos > 0 {
			name = entry[0:pos]
		} else {
			name = entry
		}
		name = strings.TrimFunc(name, isWordDelimiter)
		if len(name) > 1 {
			names[name] = true
		}

		// Extract locations.
		locationString := ""
		if pos > 0 && pos < len(entry)-1 {
			locationString = entry[pos+1:]
			locationString = strings.TrimFunc(locationString, isWordDelimiter)
			if len(locationString) > 1 {
				locs := extractTokens(locationString)
				locs = normalizeTokens(locs)
				for _, location := range locs {
					locations[location] = true
				}
			}
		}
	}
	resultNames := make([]string, 0, len(names))
	for name, _ := range names {
		resultNames = append(resultNames, name)
	}
	resultLocs := make([]string, 0, len(locations))
	for location, _ := range locations {
		resultLocs = append(resultLocs, location)
	}
	return Ancestry{line: line, Words: words, Tokens: tokens, Names: resultNames, Locations: resultLocs}
}

// Contains checks if the Ancestry contains name.
// The method checks words and tokens.
func (a *Ancestry) Contains(name string) bool {
	name = strings.ToLower(name)
	for _, n := range a.Words {
		if n == name {
			return true
		}
	}
	for _, l := range a.Tokens {
		if l == name {
			return true
		}
	}
	return false
}

// normalizeTokens brings the given token to a normalized form.
// Abbreviations are expanded, some words are translated into English,
// and junk is thrown away. The tokens should be converted to lower case
// before calling this function.
func normalizeTokens(tokens []string) []string {
	result := make([]string, 0, len(tokens))
	dirtyTags := map[string][]string{
		"gt":                       {}, // part of &gt;
		"amp":                      {}, // part of &amp;
		"ii":                       {},
		"???":                      {},
		"now":                      {},
		"also":                     {},
		"unknown":                  {},
		"al":                       {"alabama", "usa"},
		"ak":                       {"alaska", "usa"},
		"ar":                       {"arkansas", "usa"},
		"az":                       {"arizona", "usa"},
		"ca":                       {"california", "usa"},
		"co":                       {"colorado", "usa"},
		"ct":                       {"connecticut", "usa"},
		"de":                       {"delaware", "usa"},
		"dc":                       {"district of columbia", "usa"},
		"fl":                       {"florida", "usa"},
		"ga":                       {"georgia usa", "usa"},
		"hi":                       {"hawaii", "usa"},
		"ia":                       {"iowa", "usa"},
		"id":                       {"idaho", "usa"},
		"il":                       {"illinois", "usa"},
		"in":                       {"indiana", "usa"},
		"ky":                       {"kentucky", "usa"},
		"ks":                       {"kansas", "usa"},
		"la":                       {"louisiana", "usa"},
		"ma":                       {"massachusetts", "usa"},
		"md":                       {"maryland", "usa"},
		"me":                       {"maine", "usa"},
		"mi":                       {"michigan", "usa"},
		"mo":                       {"missouri", "usa"},
		"mn":                       {"minnesota", "usa"},
		"ms":                       {"mississippi", "usa"},
		"mt":                       {"montana", "usa"},
		"nc":                       {"north carolina", "usa"},
		"nd":                       {"north dakota", "usa"},
		"ne":                       {"nebraska", "usa"},
		"nh":                       {"new hampshire", "usa"},
		"nj":                       {"new jersey", "usa"},
		"nm":                       {"new mexico", "usa"},
		"nv":                       {"nevada", "usa"},
		"ny":                       {"new york", "usa"},
		"nyc":                      {"new york", "usa"},
		"oh":                       {"ohio", "usa"},
		"ok":                       {"oklahoma", "usa"},
		"or":                       {"oregon", "usa"},
		"pa":                       {"pennsylvania", "usa"},
		"ri":                       {"rhode island", "usa"},
		"sc":                       {"south carolina", "usa"},
		"sd":                       {"south dakota", "usa"},
		"tn":                       {"tennessee", "usa"},
		"tx":                       {"texas", "usa"},
		"uk":                       {"united kingdom"},
		"us":                       {"usa"},
		"ut":                       {"utah", "usa"},
		"va":                       {"virginia", "usa"},
		"vt":                       {"vermont", "usa"},
		"wa":                       {"washington", "usa"},
		"wi":                       {"wisconsin", "usa"},
		"wv":                       {"west virginia", "usa"},
		"wy":                       {"wyoming", "usa"},
		"danmark":                  {"denmark"},
		"deutschland":              {"germany"},
		"pommern":                  {"pomerania"},
		"preußen":                  {"prussia"},
		"preussen":                 {"prussia"},
		"westpreussen":             {"west prussia"},
		"russian federation":       {"russia"},
		"united states of america": {"usa"},
		"united states":            {"usa"},
		"vorpommern":               {"western pomerania"},
		"w virginia":               {"west virginia", "usa"},
	}
	for _, token := range tokens {
		// Check if token matches dirty tags.
		if clean, ok := dirtyTags[token]; ok {
			result = append(result, clean...)
		} else {
			// Check each single word of token for dirty matches.
			words := strings.FieldsFunc(token, isWordDelimiter)
			cleanWords := make([]string, 0, len(words))
			for _, word := range words {
				if clean, ok := dirtyTags[word]; ok {
					if len(clean) == 1 {
						cleanWords = append(cleanWords, clean...)
					} else if len(clean) > 1 {
						// Word has been substituted by several tokens.
						result = append(result, clean...)
					}
				} else {
					cleanWords = append(cleanWords, word)
				}
			}
			cleanToken := strings.Join(cleanWords, " ")
			if len(cleanToken) > 0 {
				result = append(result, cleanToken)
			}
		}
	}
	return result
}

// extractTokens extracts tokens from a line of text.
// A token is a collection of words that is perceived as
// a semantical unit, for example: United States of America.
// White spaces at the beginning and end of each token
// are truncated.
func extractTokens(line string) []string {
	fields := strings.FieldsFunc(line, isTokenDelimiter)
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimFunc(field, isWordDelimiter)
		if len(field) > 1 {
			result = append(result, field)
		}
	}
	return result
}

func isTokenDelimiter(c rune) bool {
	if tokenDelimiters[c] {
		return true
	} else {
		return false
	}
}

// extractWords returns the words from a line of text.
// White spaces at the beginning and end of each word
// are truncated.
func extractWords(line string) []string {
	fields := strings.FieldsFunc(line, isWordDelimiter)
	result := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if len(field) > 1 {
			result = append(result, field)
		}
	}
	return result
}

func isWordDelimiter(c rune) bool {
	if wordDelimiters[c] || tokenDelimiters[c] {
		return true
	} else {
		return false
	}
}

// Ancestries provides a convenient list for the Ancestry type.
type Ancestries []Ancestry

// NewAncestries creates Ancestries from a Family Finder matches file.
// The file must be in CSV format. namesCol is the number of the column
// that contains the ancestral informations.
func NewAncestries(filename string, namesCol int) (Ancestries, error) {
	infile, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer infile.Close()

	// Read all CSV records from file.
	csvReader := csv.NewReader(infile)
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, err
	}

	// Extract specified column.
	column := make([]string, len(records))
	for i := 0; i < len(records); i++ {
		column[i] = records[i][namesCol]
	}

	// Drop column caption.
	if len(column) > 0 {
		column = column[1:]
	} else {
		return nil, errors.New("empty file")
	}

	// Parse column.
	result := make([]Ancestry, len(column))
	for i, line := range column {
		result[i] = NewAncestry(line)
	}
	return result, nil
}

// Names returns a list of all ancestral surnames.
// Double entries are eliminated.
func (a *Ancestries) Names() []string {
	return a.eliminateDoublesIn(func(a Ancestry) []string { return a.Names })
}

// Names returns a list of all ancestral locations.
// Double entries are eliminated.
func (a *Ancestries) Locations() []string {
	return a.eliminateDoublesIn(func(a Ancestry) []string { return a.Locations })
}

// eliminateDoublesIn eliminates all double entries from a
// specified field of Ancestries.
// accFunc returns the field that contains the data.
func (a *Ancestries) eliminateDoublesIn(accFunc func(Ancestry) []string) []string {
	// Eliminate double entries.
	names := make(map[string]bool)
	for _, ancestry := range *a {
		for _, token := range accFunc(ancestry) {
			names[token] = true
		}
	}
	// Convert to array because we may need to sort it later.
	result := make([]string, 0, len(names))
	for name, _ := range names {
		result = append(result, name)
	}
	return result
}

// FrequenciesOf determines the Frequencies of the specified words.
// The words are compared against the Ancestries Word field.
func (a *Ancestries) FrequenciesOf(words []string) Frequencies {
	return a.frequenciesOf(words, func(a Ancestry) []string { return a.Words })
}

// FrequenciesOfLocations determines how many cousins share which
// ancestral locations.
func (a *Ancestries) FrequenciesOfLocations() Frequencies {
	return a.frequenciesOf(a.Locations(), func(a Ancestry) []string { return a.Locations })
}

// FrequenciesOfNames determines how many cousins share which
// ancestral surnames.
func (a *Ancestries) FrequenciesOfNames() Frequencies {
	return a.frequenciesOf(a.Names(), func(a Ancestry) []string { return a.Names })
}

// frequenciesOf calculates the Frequencies of the specified list of names.
// The access function accFunc determines which field of Ancestries should
// be used for the calculation.
func (a *Ancestries) frequenciesOf(names []string, accFunc func(Ancestry) []string) Frequencies {
	result := make([]Frequency, 0, len(names))
	for _, name := range names {
		count := 0
		for _, ancestry := range *a {
			for _, token := range accFunc(ancestry) {
				if token == strings.ToLower(name) {
					count++
					break
				}
			}
		}
		if count > 0 {
			result = append(result, Frequency{NCousins: count, Name: name})
		}
	}
	return result
}

// Filter returns only Ancestries which contain the specified name.
// The name may be a location or a surname.
func (a *Ancestries) Filter(name string) Ancestries {
	result := make([]Ancestry, 0, len(*a))
	for _, ancestry := range *a {
		if ancestry.Contains(name) {
			result = append(result, ancestry)
		}
	}
	return result
}

// Frequency shows the amount of cousins who share
// a specific ancestral surname or location.
type Frequency struct {
	NCousins int
	// Name is the ancestral name or location.
	Name string
}

// Frequencies is a list of Frequency that satisfies the sort.Interface.
type Frequencies []Frequency

func (f *Frequencies) Len() int {
	return len(*f)
}

func (f *Frequencies) Less(i, j int) bool {
	if (*f)[i].NCousins < (*f)[j].NCousins {
		return true
	} else {
		return false
	}
}

func (f *Frequencies) Swap(i, j int) {
	(*f)[i], (*f)[j] = (*f)[j], (*f)[i]
}
