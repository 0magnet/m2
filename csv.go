// Package main csv.go
package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"log"
	"strings"
	"sync"

	p "github.com/0magnet/m2/pkg/product"
	"github.com/bitfield/script"
)

var (
	allproducts   p.Products
	allproductsMu sync.RWMutex
)

func readproductscsv(csvFile string) (data []byte) {
	data, err := script.File(csvFile).Bytes() //nolint
	if err != nil {
		log.Printf(`Error reading %s file %v`, csvFile, err)
	}
	return data
}

const csvMinFields = 51 // f[0] through f[50]

// readCSV reads the catalog from a file. The parsing is in parseCSV so that
// it can be tested without one.
func readCSV(csvFile string) p.Products {
	return parseCSV(readproductscsv(csvFile))
}

// parseCSV turns the catalog bytes into products, skipping rows that are not
// enabled and rows too short to fill one.
func parseCSV(data []byte) (prods p.Products) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		f := strings.Split(line, ",")
		if len(f) < 4 {
			continue
		}
		if f[3] != "TRUE" {
			continue
		}
		if len(f) < csvMinFields {
			log.Printf("csv line %d: expected %d fields, got %d — skipping", lineNum, csvMinFields, len(f))
			continue
		}
		q := p.Product{
			Image1:            f[0],
			Partno:            f[1],
			Name:              f[2],
			Enable:            f[3],
			Price:             f[4],
			Quantity:          f[5],
			Shippable:         f[6],
			Minorder:          f[7],
			Maxorder:          f[8],
			Defaultquantity:   f[9],
			Stepquantity:      f[10],
			Mfgpartno:         f[11],
			Mfgname:           f[12],
			Category:          f[13],
			Subcategory:       f[14],
			Location:          f[15],
			Msrp:              f[16],
			Cost:              f[17],
			Typ:               f[18],
			Packagetype:       f[19],
			Technology:        f[20],
			Materials:         f[21],
			Value:             f[22],
			ValUnit:           f[23],
			Resistance:        f[24],
			ResUnit:           f[25],
			Tolerance:         f[26],
			VoltsRating:       f[27],
			AmpsRating:        f[28],
			WattsRating:       f[29],
			TempRating:        f[30],
			TempUnit:          f[31],
			Description1:      f[32],
			Description2:      f[33],
			Color1:            f[34],
			Color2:            f[35],
			Sourceinfo:        f[36],
			Datasheet:         f[37],
			Docs:              f[38],
			Reference:         f[39],
			Attributes:        f[40],
			Year:              f[41],
			Condition:         f[42],
			Note:              f[43],
			Warning:           f[44],
			CableLengthInches: f[45],
			LengthInches:      f[46],
			WidthInches:       f[47],
			HeightInches:      f[48],
			WeightLb:          f[49],
			WeightOz:          f[50],
		}
		prods = append(prods, q)
	}
	return prods
}

// validateCSV scans products for patterns that could cause issues in HTML/JS rendering.
// Returns a list of warnings. Call after readCSV to check data integrity.
func validateCSV(prods p.Products) []string {
	var warnings []string
	for i, pr := range prods {
		check := func(field, value string) {
			if strings.ContainsAny(value, "<>\"'&") {
				warnings = append(warnings, fmt.Sprintf("product %d (%s): %s contains HTML-unsafe characters: %q", i, pr.Partno, field, value))
			}
			if strings.Contains(value, "|") {
				warnings = append(warnings, fmt.Sprintf("product %d (%s): %s contains pipe character: %q", i, pr.Partno, field, value))
			}
		}
		check("Partno", pr.Partno)
		check("Name", pr.Name)
		check("Description1", pr.Description1)
		check("Description2", pr.Description2)
		check("Note", pr.Note)
		check("Warning", pr.Warning)
		check("Category", pr.Category)
		check("Subcategory", pr.Subcategory)
		check("Mfgname", pr.Mfgname)
		check("Mfgpartno", pr.Mfgpartno)
	}
	return warnings
}
