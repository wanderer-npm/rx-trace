package scrapers

import (
	"encoding/json"
)

type fdaEnvelope struct {
	Results []fdaRecord `json:"results"`
}

type fdaRecord struct {
	ProprietaryName    string  `json:"proprietary_name"`
	GenericName        string  `json:"generic_name"`
	Status             string  `json:"status"`
	UpdateType         string  `json:"update_type"`
	InitialPostingDate string  `json:"initial_posting_date"`
	UpdateDate         string  `json:"update_date"`
	Reason             string  `json:"reason"`
	Availability       string  `json:"availability"`
	OpenFDA            openFDA `json:"openfda"`
}

type openFDA struct {
	BrandName   []string `json:"brand_name"`
	GenericName []string `json:"generic_name"`
}

type overpassEnvelope struct {
	Elements []overpassEl `json:"elements"`
}

type overpassEl struct {
	Type   string            `json:"type"`
	ID     int64             `json:"id"`
	Lat    float64           `json:"lat"`
	Lon    float64           `json:"lon"`
	Center *overpassCenter   `json:"center,omitempty"`
	Tags   map[string]string `json:"tags"`
}

type overpassCenter struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

func decode(b []byte, v any) error {
	return json.Unmarshal(b, v)
}
