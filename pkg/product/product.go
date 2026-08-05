// Package product pkg/product/product.go
package product

type Product struct {
	Enable            string
	Partno            string
	Name              string
	Image1            string
	Price             string
	Quantity          string
	Shippable         string
	Minorder          string
	Maxorder          string
	Defaultquantity   string
	Stepquantity      string
	Mfgpartno         string
	Mfgname           string
	Category          string
	Subcategory       string
	Location          string
	Msrp              string
	Cost              string
	Typ               string
	Packagetype       string
	Technology        string
	Materials         string
	Value             string
	ValUnit           string
	Resistance        string
	ResUnit           string
	Tolerance         string
	VoltsRating       string
	AmpsRating        string
	WattsRating       string
	TempRating        string
	TempUnit          string
	Description1      string
	Description2      string
	Color1            string
	Color2            string
	Sourceinfo        string
	Datasheet         string
	Docs              string
	Reference         string
	Attributes        string
	Year              string
	Condition         string
	Note              string
	Warning           string
	CableLengthInches string
	LengthInches      string
	WidthInches       string
	HeightInches      string
	WeightLb          string
	WeightOz          string
}

// Products is an array of Product
type Products []Product
