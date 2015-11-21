// Package cousins provides classes to work with cousins' ancestral information.
package cousins

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"unicode"
)

// minTokenLen is the minimal length of a token.
const minTokenLen = 2

// tokenDelimiters separate semantical units of text. Each token delimiter is also a word delimiter.
var tokenDelimiters = map[rune]bool{',': true, '/': true, '(': true, ')': true, '-': true, '&': true, ';': true}

// Ancestry contains one cousin's ancestral surnames and locations.
type Ancestry struct {
	// line is the original line from the Family Finder matches file
	// in small caps.
	line string
	// Words are the different words contained in the line.
	Words map[string]bool
	// Tokens are the different tokens contained in the line.
	// A token consists of one or more words that belong together sematically,
	// for example "United States of America".
	Tokens map[string]bool
	// Names are the different ancestral surnames.
	Names     map[string]bool
	Locations map[string]bool
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
				for loc, _ := range locs {
					locations[loc] = true
				}
			}
		}
	}
	return Ancestry{line: line, Words: words, Tokens: tokens, Names: names, Locations: locations}
}

// Contains checks if the Ancestry contains name.
// The method checks words and tokens.
func (a *Ancestry) Contains(name string) bool {
	name = strings.ToLower(name)
	return a.Words[name] || a.Tokens[name]
}

// normalizeTokens transforms the given tokens into a normalized form.
// Abbreviations are expanded, some words are translated into English,
// and junk is thrown away. The tokens should be converted to lower case
// before calling this function.
func normalizeTokens(tokens map[string]bool) map[string]bool {
	result := make(map[string]bool)

	// dirtyTags is a map of tokens that are transformed
	// into the normalized form.
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
		"alabama":                  {"alabama", "usa"},
		"alaska":                   {"alaska", "usa"},
		"arkansas":                 {"arkansas", "usa"},
		"arizona":                  {"arizona", "usa"},
		"california":               {"california", "usa"},
		"colorado":                 {"colorado", "usa"},
		"connecticut":              {"connecticut", "usa"},
		"delaware":                 {"delaware", "usa"},
		"district of columbia":     {"district of columbia", "usa"},
		"florida":                  {"florida", "usa"},
		"georgia usa":              {"georgia usa", "usa"},
		"hawaii":                   {"hawaii", "usa"},
		"iowa":                     {"iowa", "usa"},
		"idaho":                    {"idaho", "usa"},
		"illinois":                 {"illinois", "usa"},
		"indiana":                  {"indiana", "usa"},
		"kentucky":                 {"kentucky", "usa"},
		"kansas":                   {"kansas", "usa"},
		"louisiana":                {"louisiana", "usa"},
		"massachusetts":            {"massachusetts", "usa"},
		"maryland":                 {"maryland", "usa"},
		"maine":                    {"maine", "usa"},
		"michigan":                 {"michigan", "usa"},
		"missouri":                 {"missouri", "usa"},
		"minnesota":                {"minnesota", "usa"},
		"mississippi":              {"mississippi", "usa"},
		"montana":                  {"montana", "usa"},
		"north carolina":           {"north carolina", "usa"},
		"north dakota":             {"north dakota", "usa"},
		"nebraska":                 {"nebraska", "usa"},
		"new hampshire":            {"new hampshire", "usa"},
		"new jersey":               {"new jersey", "usa"},
		"new mexico":               {"new mexico", "usa"},
		"nevada":                   {"nevada", "usa"},
		"new york":                 {"new york", "usa"},
		"ohio":                     {"ohio", "usa"},
		"oklahoma":                 {"oklahoma", "usa"},
		"oregon":                   {"oregon", "usa"},
		"pennsylvania":             {"pennsylvania", "usa"},
		"rhode island":             {"rhode island", "usa"},
		"south carolina":           {"south carolina", "usa"},
		"south dakota":             {"south dakota", "usa"},
		"tennessee":                {"tennessee", "usa"},
		"texas":                    {"texas", "usa"},
		"utah":                     {"utah", "usa"},
		"virginia":                 {"virginia", "usa"},
		"vermont":                  {"vermont", "usa"},
		"washington":               {"washington", "usa"},
		"wisconsin":                {"wisconsin", "usa"},
		"west virginia":            {"west virginia", "usa"},
		"wyoming":                  {"wyoming", "usa"},
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
	for token, _ := range tokens {
		// Check if token matches dirty tags.
		if cleanTokens, ok := dirtyTags[token]; ok {
			for _, clean := range cleanTokens {
				result[clean] = true
			}
		} else {
			// Check each single word of token for dirty matches.
			words := strings.FieldsFunc(token, isWordDelimiter)
			cleanWords := make([]string, 0, len(words))
			for _, word := range words {
				if cleanTokens, ok := dirtyTags[word]; ok {
					if len(cleanTokens) == 1 {
						cleanWords = append(cleanWords, cleanTokens...)
					} else if len(cleanTokens) > 1 {
						// Word has been substituted by several tokens.
						for _, clean := range cleanTokens {
							result[clean] = true
						}
					}
				} else {
					cleanWords = append(cleanWords, word)
				}
			}
			cleanToken := strings.Join(cleanWords, " ")
			if len(cleanToken) >= minTokenLen {
				result[cleanToken] = true
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
func extractTokens(line string) map[string]bool {
	fields := strings.FieldsFunc(line, isTokenDelimiter)
	result := make(map[string]bool)
	for _, field := range fields {
		field = strings.TrimFunc(field, isWordDelimiter)
		if len(field) >= minTokenLen {
			result[field] = true
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
func extractWords(line string) map[string]bool {
	fields := strings.FieldsFunc(line, isWordDelimiter)
	result := make(map[string]bool)
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if len(field) >= minTokenLen {
			result[field] = true
		}
	}
	return result
}

func isWordDelimiter(c rune) bool {
	if unicode.IsSpace(c) || tokenDelimiters[c] {
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
	// Read whole file.
	inbytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	// Check if file contains UTF8 byte order mark and drop it if necessary.
	var dataReader *bytes.Reader
	BOM := []byte{0xEF, 0xBB, 0xBF}
	if bytes.Compare(inbytes[:3], BOM) == 0 {
		dataReader = bytes.NewReader(inbytes[3:])
	} else {
		dataReader = bytes.NewReader(inbytes)
	}

	// Read all CSV records from UTF8 buffer.
	csvReader := csv.NewReader(dataReader)
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

// Names returns a set of all ancestral surnames.
func (a *Ancestries) Names() map[string]bool {
	result := make(map[string]bool)
	for _, ancestry := range *a {
		for name, _ := range ancestry.Names {
			result[name] = true
		}
	}
	return result
}

// Locations returns a set of all ancestral locations.
func (a *Ancestries) Locations() map[string]bool {
	result := make(map[string]bool)
	for _, ancestry := range *a {
		for loc, _ := range ancestry.Locations {
			result[loc] = true
		}
	}
	return result
}

// FrequenciesOf determines the Frequencies of the specified words.
// The words are compared against the Ancestries Word field.
func (a *Ancestries) FrequenciesOf(words map[string]bool) Frequencies {
	return a.frequenciesOf(words, func(anc Ancestry) map[string]bool { return anc.Words })
}

// FrequenciesOfLocations determines how many cousins share which
// ancestral locations.
func (a *Ancestries) FrequenciesOfLocations() Frequencies {
	return a.frequenciesOf(a.Locations(), func(anc Ancestry) map[string]bool { return anc.Locations })
}

// FrequenciesOfNames determines how many cousins share which
// ancestral surnames.
func (a *Ancestries) FrequenciesOfNames() Frequencies {
	return a.frequenciesOf(a.Names(), func(anc Ancestry) map[string]bool { return anc.Names })
}

// frequenciesOf calculates the Frequencies of the specified set of names.
// The access function accFunc determines which field of Ancestries should
// be used for the calculation.
func (a *Ancestries) frequenciesOf(names map[string]bool, accFunc func(Ancestry) map[string]bool) Frequencies {
	result := make([]Frequency, 0, len(names))
	for name, _ := range names {
		count := 0
		for _, ancestry := range *a {
			namesInAnc := accFunc(ancestry)
			if namesInAnc[strings.ToLower(name)] {
				count++
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

// Exclude returns only those Ancestries who's ancestral surnames
// or locations do not contain name.")
func (a *Ancestries) Exclude(name string) Ancestries {
	result := make([]Ancestry, 0, len(*a))
	for _, ancestry := range *a {
		if !ancestry.Contains(name) {
			result = append(result, ancestry)
		}
	}
	return result
}

// Frequency shows the amount of cousins who share
// a specific ancestral surname or location.
type Frequency struct {
	// Name is the ancestral name or location.
	Name string
	// NCousins shows how many cousins share the same ancestral name or location.
	NCousins int
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

// WriteCSV writes the frequencies to a file as comma separated values.
// It adds an header containing the captions "Location" and "Value".
// Frequencies where NCousins == 0 are not written to file.
func (f *Frequencies) WriteCSV(filename string) error {
	// Open file.
	outfile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer outfile.Close()

	// Write header.
	writer := bufio.NewWriter(outfile)
	_, err = writer.WriteString(fmt.Sprintf("%s,%s\r\n", "Location", "Value"))
	if err != nil {
		return err
	}
	// Write rows.
	for _, freq := range *f {
		if freq.NCousins > 0 {
			_, err = writer.WriteString(fmt.Sprintf("%s,%d\r\n", freq.Name, freq.NCousins))
			if err != nil {
				return err
			}
		}
	}
	err = writer.Flush()
	return err
}
