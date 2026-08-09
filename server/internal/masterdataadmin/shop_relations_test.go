package masterdataadmin

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestLoadAddsShopCellRelations(t *testing.T) {
	path := filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e")
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		t.Skip("repository master data is not installed")
	} else if err != nil {
		t.Fatal(err)
	}
	catalog, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	shop := findCatalogTable(catalog, "m_shop")
	foundShopRelation := false
	for _, row := range shop.Rows {
		if len(row.ShopRelations) == 0 {
			continue
		}
		relation := row.ShopRelations[0]
		shopID, err := strconv.ParseInt(identityValue(row, "ShopId"), 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		cellGroupID, err := strconv.ParseInt(identityValue(row, "ShopItemCellGroupId"), 10, 64)
		if err != nil {
			t.Fatal(err)
		}
		if relation.ShopID != shopID || relation.ShopItemCellGroupID != cellGroupID || len(relation.ShopItemCellIDs) == 0 {
			t.Fatalf("shop relation does not match row identity: %+v", relation)
		}
		foundShopRelation = true
		break
	}
	if !foundShopRelation {
		t.Fatal("no shop row has a cell relation")
	}

	terms := findCatalogTable(catalog, "m_shop_item_cell_term")
	for _, row := range terms.Rows {
		if len(row.ShopRelations) == 0 {
			continue
		}
		for _, relation := range row.ShopRelations {
			if relation.ShopID == 0 || relation.ShopItemCellGroupID == 0 || len(relation.ShopItemCellIDs) == 0 {
				t.Fatalf("incomplete term relation: %+v", relation)
			}
		}
		return
	}
	t.Fatal("no shop item cell term has a shop relation")
}

func findCatalogTable(catalog *Catalog, name string) Table {
	for _, table := range catalog.Tables {
		if table.Name == name {
			return table
		}
	}
	return Table{}
}

func identityValue(row Row, name string) string {
	for _, field := range row.Identity {
		if field.Name == name {
			return field.Value
		}
	}
	return row.Values[name]
}
