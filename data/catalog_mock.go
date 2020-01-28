package data

/*
 Copyright 2019 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at
      http://www.apache.org/licenses/LICENSE-2.0
 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

import (
	"bytes"
	"fmt"
	"strconv"
	"text/template"

	log "github.com/sirupsen/logrus"
)

type catalogMock struct {
	tables    []*Table
	tableData map[string][]*featureMock
	functions []*Function
}

var instance Catalog

// CatMockInstance tbd
func CatMockInstance() Catalog {
	log.Printf("Using Test Catalog data")
	// TODO: make a singleton
	instance = newCatalogMock()
	return instance
}

var templateFeaturePoint *template.Template

func init() {
	templateFeaturePoint = template.Must(template.New("feature").Parse(templateStrFeaturePoint))
}

func newCatalogMock() Catalog {
	// must be in synch with featureMock type
	propNames := []string{"propA", "propB"}
	types := map[string]string{
		"propA": "text",
		"propB": "int"}
	jtypes := []string{"string", "number"}

	layerA := &Table{
		ID:          "mock_a",
		Title:       "Mock A",
		Description: "This dataset contains mock data about A (9 points)",
		Extent:      Extent{Minx: -120, Miny: 40, Maxx: -74, Maxy: 50},
		Srid:        4326,
		Columns:     propNames,
		Types:       types,
		JSONTypes:   jtypes,
	}

	layerB := &Table{
		ID:          "mock_b",
		Title:       "Mock B",
		Description: "This dataset contains mock data about B (100 points)",
		Extent:      Extent{Minx: -75, Miny: 45, Maxx: -74, Maxy: 46},
		Srid:        4326,
		Columns:     propNames,
		Types:       types,
		JSONTypes:   jtypes,
	}

	layerC := &Table{
		ID:          "mock_c",
		Title:       "Mock C",
		Description: "This dataset contains mock data about C (10000 points)",
		Extent:      Extent{Minx: -120, Miny: 40, Maxx: -74, Maxy: 60},
		Srid:        4326,
		Columns:     propNames,
		Types:       types,
		JSONTypes:   jtypes,
	}

	tableData := map[string][]*featureMock{}
	tableData["mock_a"] = makePointFeatures(layerA.Extent, 3, 3)
	tableData["mock_b"] = makePointFeatures(layerB.Extent, 10, 10)
	tableData["mock_c"] = makePointFeatures(layerC.Extent, 100, 100)

	var tables []*Table
	tables = append(tables, layerA)
	tables = append(tables, layerB)
	tables = append(tables, layerC)

	catMock := catalogMock{
		tables:    tables,
		tableData: tableData,
	}

	return &catMock
}

func (cat *catalogMock) Tables() ([]*Table, error) {
	return cat.tables, nil
}

func (cat *catalogMock) TableByName(name string) (*Table, error) {
	for _, lyr := range cat.tables {
		if lyr.ID == name {
			return lyr, nil
		}
	}
	// not found - indicated by nil value returned
	return nil, nil
}

func (cat *catalogMock) TableFeatures(name string, param QueryParam) ([]string, error) {
	features, ok := cat.tableData[name]
	if !ok {
		// table not found - indicated by nil value returned
		return nil, nil
	}
	start := 0
	end := len(features) - 1
	if param.Limit < len(features) {
		start = param.Offset
		end = param.Offset + param.Limit - 1
		if end >= len(features) {
			end = len(features) - 1
		}
	}
	propNames := cat.tables[0].Columns
	if len(param.Properties) > 0 {
		propNames = param.Properties
	}
	return featuresToJSON(features, start, end, propNames), nil
}

func (cat *catalogMock) TableFeature(name string, id string, param QueryParam) (string, error) {
	features, ok := cat.tableData[name]
	if !ok {
		// table not found - indicated by empty value returned
		return "", nil
	}
	index, err := strconv.Atoi(id)
	if err != nil {
		// a malformed int is treated as feature not found
		return "", nil
	}

	// TODO: return not found if index out of range
	if index < 0 || index >= len(features) {
		return "", nil
	}
	propNames := cat.tables[0].Columns
	if len(param.Properties) > 0 {
		propNames = param.Properties
	}

	return features[index].toJSON(propNames), nil
}

func (cat *catalogMock) Functions() ([]*Function, error) {
	return cat.functions, nil
}

func (cat *catalogMock) FunctionByName(name string) (*Function, error) {
	for _, fn := range cat.functions {
		if fn.ID == name {
			return fn, nil
		}
	}
	// not found - indicated by nil value returned
	return nil, nil
}

func (cat *catalogMock) FunctionFeatures(name string, param QueryParam) ([]string, error) {
	// TODO:
	return nil, nil
}

func (cat *catalogMock) FunctionData(name string, param QueryParam) ([]map[string]interface{}, error) {
	// TODO:
	return nil, nil
}

func makePointFeatures(extent Extent, nx int, ny int) []*featureMock {
	basex := extent.Minx
	basey := extent.Miny
	dx := (extent.Maxx - extent.Minx) / float64(nx)
	dy := (extent.Maxy - extent.Miny) / float64(ny)

	n := nx * ny
	features := make([]*featureMock, n)
	index := 0
	for ix := 0; ix < nx; ix++ {
		for iy := 0; iy < ny; iy++ {
			id := index + 1
			x := basex + dx*float64(ix)
			y := basey + dy*float64(iy)
			features[index] = makeFeatureMockPoint(id, x, y)
			//fmt.Println(features[index])

			index++
		}
	}
	return features
}

type featurePointMock struct {
	ID  int
	X   float64
	Y   float64
	Val string
}

var templateStrFeaturePoint = `{ "type": "Feature", "id": "{{ .ID }}",
"geometry": {"type": "Point","coordinates": [  {{ .X }}, {{ .Y }} ]  },
"properties": { "value": "{{ .Val }}"  } }`

func makeFeaturePoint(id int, x float64, y float64, val string) string {
	feat := featurePointMock{id, x, y, val}
	var tempOut bytes.Buffer
	//	tempOut.Reset()
	templateFeaturePoint.Execute(&tempOut, feat)
	return tempOut.String()
}

type featureMock struct {
	ID    string
	Geom  string
	PropA string
	PropB int
}

func makeFeatureMockPoint(id int, x float64, y float64) *featureMock {
	geomFmt := `{"type": "Point","coordinates": [ %v, %v ]  }`
	geomStr := fmt.Sprintf(geomFmt, x, y)

	idstr := strconv.Itoa(id)
	feat := featureMock{idstr, geomStr, "propA", id}
	return &feat
}

func (fm *featureMock) toJSON(propNames []string) string {
	props := fm.extractProperties(propNames)
	return makeFeatureJSON(fm.ID, fm.Geom, props)
}

func (fm *featureMock) extractProperties(propNames []string) map[string]interface{} {
	isAll := len(propNames) == 0
	props := make(map[string]interface{})
	for _, name := range propNames {
		if isAll || name == "propA" {
			props[name] = fm.PropA
		}
		if isAll || name == "propB" {
			props[name] = fm.PropB
		}
	}
	return props
}

func featuresToJSON(features []*featureMock, start int, end int, propNames []string) []string {
	n := end - start + 1
	featJSON := make([]string, n)
	for i := 0; i < n; i++ {
		featJSON[i] = features[start+i].toJSON(propNames)
	}
	return featJSON
}
