package main

import (
	"strings"
	"testing"
)

// row builds a catalog line with the fields the parser needs, so a test can
// set the few it cares about without writing fifty-one commas.
func row(set map[int]string) string {
	f := make([]string, csvMinFields)
	f[0] = "img.jpg"
	f[1] = "PN-1"
	f[2] = "A Part"
	f[3] = "TRUE"
	f[4] = "1.50"
	f[5] = "10"
	for i, v := range set {
		f[i] = v
	}
	return strings.Join(f, ",")
}

func TestParseCSVReadsAProduct(t *testing.T) {
	prods := parseCSV([]byte(row(nil)))
	if len(prods) != 1 {
		t.Fatalf("got %d products, want 1", len(prods))
	}
	got := prods[0]
	if got.Partno != "PN-1" || got.Name != "A Part" || got.Price != "1.50" || got.Quantity != "10" {
		t.Errorf("fields did not land where they should: %+v", got)
	}
}

// The last field is the fiftieth, and off-by-one at the end of a fifty-one
// column row is invisible in a spreadsheet.
func TestParseCSVReadsTheLastField(t *testing.T) {
	prods := parseCSV([]byte(row(map[int]string{50: "3.25", 49: "1"})))
	if len(prods) != 1 {
		t.Fatalf("got %d products, want 1", len(prods))
	}
	if prods[0].WeightOz != "3.25" {
		t.Errorf("WeightOz = %q, want the last column", prods[0].WeightOz)
	}
	if prods[0].WeightLb != "1" {
		t.Errorf("WeightLb = %q, want the second to last column", prods[0].WeightLb)
	}
}

// Only enabled rows are sold. A row that says anything but TRUE is a product
// deliberately withdrawn, and showing it anyway sells something not in stock.
func TestParseCSVSkipsRowsThatAreNotEnabled(t *testing.T) {
	for _, enable := range []string{"FALSE", "false", "true", "", "1", "yes"} {
		prods := parseCSV([]byte(row(map[int]string{3: enable})))
		if len(prods) != 0 {
			t.Errorf("Enable=%q produced a product; only TRUE should", enable)
		}
	}
}

// A short row cannot fill a product, and reading one anyway would panic on a
// live catalog rather than skip a line.
func TestParseCSVSkipsShortRows(t *testing.T) {
	short := strings.Join([]string{"img.jpg", "PN-1", "A Part", "TRUE", "1.50"}, ",")
	if prods := parseCSV([]byte(short)); len(prods) != 0 {
		t.Errorf("a row with 5 fields produced %d products", len(prods))
	}
	// Exactly one short of the minimum is the boundary worth pinning.
	f := make([]string, csvMinFields-1)
	for i := range f {
		f[i] = "x"
	}
	f[3] = "TRUE"
	if prods := parseCSV([]byte(strings.Join(f, ","))); len(prods) != 0 {
		t.Errorf("a row one field short produced %d products", len(prods))
	}
}

func TestParseCSVSkipsVeryShortAndBlankRows(t *testing.T) {
	for _, line := range []string{"", ",", "a,b,c", "\n\n"} {
		if prods := parseCSV([]byte(line)); len(prods) != 0 {
			t.Errorf("%q produced %d products", line, len(prods))
		}
	}
}

func TestParseCSVReadsSeveralRows(t *testing.T) {
	data := strings.Join([]string{
		row(map[int]string{1: "AAA"}),
		row(map[int]string{3: "FALSE", 1: "SKIPPED"}),
		row(map[int]string{1: "BBB"}),
	}, "\n")
	prods := parseCSV([]byte(data))
	if len(prods) != 2 {
		t.Fatalf("got %d products, want 2", len(prods))
	}
	if prods[0].Partno != "AAA" || prods[1].Partno != "BBB" {
		t.Errorf("got %q and %q, want AAA and BBB", prods[0].Partno, prods[1].Partno)
	}
}

// The catalog is split on commas rather than parsed as CSV, so a quoted
// field containing one shifts every column after it — including Enable, which
// is column 3. The row then does not say TRUE where the parser looks, and the
// product is dropped from the store without a word.
//
// A spreadsheet writes that quoting itself the moment a name contains a comma,
// so this is reachable from ordinary editing rather than from bad data.
//
// This test records what the parser does today. If it is ever changed to use
// encoding/csv, this test is the one that should fail.
func TestParseCSVSilentlyDropsRowsWithAQuotedComma(t *testing.T) {
	f := make([]string, csvMinFields)
	for i := range f {
		f[i] = "x"
	}
	f[1] = "PN-1"
	f[2] = `"Resistor, 10k"` // one field in a spreadsheet, two after a split
	f[3] = "TRUE"
	f[4] = "1.50"
	line := strings.Join(f, ",")

	// The row is long enough — it is the shift that loses it, not the length.
	if n := len(strings.Split(line, ",")); n <= csvMinFields {
		t.Fatalf("the fixture has %d fields; it needs more than %d to isolate the shift", n, csvMinFields)
	}

	prods := parseCSV([]byte(line))
	if len(prods) != 0 {
		t.Fatalf("the parser now keeps rows with a quoted comma (%d products); "+
			"replace this test with one asserting the fields are correct", len(prods))
	}

	// Without the comma the same row is read, which is what shows the comma is
	// the cause rather than anything else in the fixture.
	f[2] = "Resistor 10k"
	if prods := parseCSV([]byte(strings.Join(f, ","))); len(prods) != 1 {
		t.Errorf("the same row without the comma gave %d products, want 1", len(prods))
	}
}

func TestValidateCSVAcceptsCleanData(t *testing.T) {
	prods := parseCSV([]byte(row(nil)))
	if w := validateCSV(prods); len(w) != 0 {
		t.Errorf("clean data produced warnings: %v", w)
	}
}

// These characters are what turn a product name into markup on the page.
func TestValidateCSVFlagsCharactersThatBreakThePage(t *testing.T) {
	for _, bad := range []string{`<script>`, `a"b`, "a'b", "a&b", "a>b"} {
		prods := parseCSV([]byte(row(map[int]string{2: bad})))
		if len(prods) != 1 {
			t.Fatalf("%q: got %d products", bad, len(prods))
		}
		w := validateCSV(prods)
		if len(w) == 0 {
			t.Errorf("%q in a product name was not flagged", bad)
			continue
		}
		if !strings.Contains(w[0], "Name") {
			t.Errorf("%q: the warning does not name the field: %s", bad, w[0])
		}
	}
}

// The pipe is the field separator in the cart's stored format, so a pipe in a
// product name corrupts the cart rather than the page.
func TestValidateCSVFlagsPipes(t *testing.T) {
	prods := parseCSV([]byte(row(map[int]string{2: "A|B"})))
	w := validateCSV(prods)
	if len(w) == 0 {
		t.Fatal("a pipe in a product name was not flagged")
	}
	if !strings.Contains(w[0], "pipe") {
		t.Errorf("the warning does not mention the pipe: %s", w[0])
	}
}

// Every field that reaches the page is checked, not just the name.
func TestValidateCSVChecksEveryRenderedField(t *testing.T) {
	fields := map[int]string{
		1:  "PN<",  // Partno
		2:  "N<",   // Name
		12: "MFG<", // Mfgname
		11: "MP<",  // Mfgpartno
		13: "CAT<", // Category
		14: "SUB<", // Subcategory
		32: "D1<",  // Description1
		33: "D2<",  // Description2
		43: "NT<",  // Note
		44: "WN<",  // Warning
	}
	for col, val := range fields {
		prods := parseCSV([]byte(row(map[int]string{3: "TRUE", col: val})))
		if len(prods) != 1 {
			t.Fatalf("column %d: got %d products", col, len(prods))
		}
		if w := validateCSV(prods); len(w) == 0 {
			t.Errorf("an unsafe character in column %d was not flagged", col)
		}
	}
}

func TestValidateCSVOnNoProducts(t *testing.T) {
	if w := validateCSV(nil); len(w) != 0 {
		t.Errorf("an empty catalog produced warnings: %v", w)
	}
}

// The warning names the row so it can be found in a spreadsheet of thousands.
func TestValidateCSVNamesTheProduct(t *testing.T) {
	prods := parseCSV([]byte(row(map[int]string{1: "PN-SPECIAL", 2: "bad<"})))
	w := validateCSV(prods)
	if len(w) == 0 {
		t.Fatal("no warning")
	}
	if !strings.Contains(w[0], "PN-SPECIAL") {
		t.Errorf("the warning does not name the part: %s", w[0])
	}
}
