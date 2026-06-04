package server

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"testing"

	"github.com/project-agonyl/agonyl-utils-go/dropfile"
	"github.com/project-agonyl/agonyl-utils-go/itemcombinationdata"
	"github.com/project-agonyl/agonyl-utils-go/itemfile"
	"github.com/project-agonyl/agonyl-utils-go/questfile"
)

var questFixtures = map[string][]byte{
	"0001.dat": mustDecodeQuestFixture("AQAAAP8DAAD/AwAA///////////////////////////////////////////////////////////////////////////////////////////AJwkAUMMAAGQAAAAAAAAABP///wEA//+Dyv//Bv////////////////////////////////////////////////////////////////////////////////////////////////////////8SAAAAdGJyZW4AAAAAAAAAAAAAAAAA/////////////v////////////////////////////////////////////////////////////////////////////////////////////////////////////8AAAAA/////////////v////////////////////////////////////////////////////////////////////////////////////////////////////////////8AAAAA/////////////v////////////////////////////////////////////////////////////////////////////////////////////////////////////8AAAAA/////////////v////////////////////////////////////////////////////////////////////////////////////////////////////////////8AAAAA/////////////v////////////////////////////////////////////////////////////////////////////////////////////////////////////8AAAAA/////////////v////////////////////////////////////////////////////////////////////////////////////////////////////////////8AAAAAAgAAAP//////////"),
	"0002.dat": mustDecodeQuestFixture("AgAAAP8DAAAABAAA//////////////////////////////////////////////////////////////////////////////////////////8goQcAUMMAAGQAAAAAAAAABP///wEA//9oyP//Bv////////////////////////////////////////////////////////////////////////////////////////////////////////8SAAAAbml0YQAAAAAAAAAAAAAAAAAA/////////////v////////////////////////////////////////////////////////////////////////////////////////////////////////////8AAAAA/////////////v////////////////////////////////////////////////////////////////////////////////////////////////////////////8AAAAA/////////////v////////////////////////////////////////////////////////////////////////////////////////////////////////////8AAAAA/////////////v////////////////////////////////////////////////////////////////////////////////////////////////////////////8AAAAA/////////////v////////////////////////////////////////////////////////////////////////////////////////////////////////////8AAAAA/////////////v////////////////////////////////////////////////////////////////////////////////////////////////////////////8AAAAAAwAAAAwAAAD/////"),
	"0412.dat": mustDecodeQuestFixture("nAEAAP8DAAD/AwAAAAAAAP/////////////////////IAAAAyAAAAP////8GIAAABSAAAAQgAAD///////////////8BAAAAAQAAAAEAAADwSQIAoIYBAFgCAAAZAAAAAP///xIA//+AgP//gP///zQA/////////////2UAAAD//////////////////////////////////////////////////////////wP///////////////////8AAAAAAf///xIA//+AgP//gP//////////////ZQD///////////////////////////////////////8FAP////////////////////////////////////////////8AAAAAAv///wEA//+Dyv//Bv////8D////////ZQD///////////////////////////////////////8FAP////////////////////////////////////////////8AAAAA/////////////v////////////////////////////////////////////////////////////////////////////////////////////////////////////8AAAAA/////////////v////////////////////////////////////////////////////////////////////////////////////////////////////////////8AAAAA/////////////v////////////////////////////////////////////////////////////////////////////////////////////////////////////8AAAAA/////////////v////////////////////////////////////////////////////////////////////////////////////////////////////////////8AAAAA////////////////"),
	"1432.dat": mustDecodeQuestFixture("mAUAAOsDAADrAwAAAAAAAP////////////////////88AAAASwAAAP////8RAAAA//////////////////////////8BAAAA///////////A4eQAQEIPANgAAAAMAAAAAP///wcA//+AgP//gP///wEA//9kAP////////////////////////////////////////////////////////////////////////////////////////////8AAAAAAP///wQA//+AgP//gP///woA//9kAP////////////////////////////////////////////////////////////////////////////////////////////8AAAAAAP///wMA//+AgP//gP///xMA//9kAP////////////////////////////////////////////////////////////////////////////////////////////8AAAAA/////////////v////////////////////////////////////////////////////////////////////////////////////////////////////////////8AAAAA/////////////v////////////////////////////////////////////////////////////////////////////////////////////////////////////8AAAAA/////////////v////////////////////////////////////////////////////////////////////////////////////////////////////////////8AAAAA/////////////v////////////////////////////////////////////////////////////////////////////////////////////////////////////8AAAAA////////////////"),
}

func TestQuestFixturesParse(t *testing.T) {
	for _, name := range questFixtureNames() {
		t.Run(name, func(t *testing.T) {
			raw := readQuestFixture(t, name)
			if _, err := questfile.Read(bytes.NewReader(raw)); err != nil {
				t.Fatalf("questfile.Read() error = %v", err)
			}
		})
	}
}

func TestQuestAPIDataRoundTripPreservesFixtureBytes(t *testing.T) {
	for _, name := range questFixtureNames() {
		t.Run(name, func(t *testing.T) {
			raw := readQuestFixture(t, name)
			qf, err := questfile.Read(bytes.NewReader(raw))
			if err != nil {
				t.Fatalf("questfile.Read() error = %v", err)
			}

			apiData := questFileToAPIData(qf)
			applyQuestFileAPIData(&qf, apiData)

			var buf bytes.Buffer
			if err := questfile.Write(&buf, qf); err != nil {
				t.Fatalf("questfile.Write() error = %v", err)
			}

			if !bytes.Equal(raw, buf.Bytes()) {
				t.Fatalf("round-trip changed fixture bytes: before=%d after=%d", len(raw), buf.Len())
			}
		})
	}
}

func TestQuestUnusedObjectiveSentinelBlock(t *testing.T) {
	raw := readQuestFixture(t, "0412.dat")
	qf, err := questfile.Read(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("questfile.Read() error = %v", err)
	}

	apiData := questFileToAPIData(qf)
	if !apiData.Objectives[3].IsUnused {
		t.Fatal("expected objective 4 to be unused")
	}

	applyQuestFileAPIData(&qf, apiData)
	block := qf.Objectives[3].Block
	for i, b := range block {
		expected := byte(0xff)
		if i == 9 {
			expected = 0xfe
		}
		if i >= 92 {
			expected = 0
		}
		if b != expected {
			t.Fatalf("unused block byte %d = 0x%02x, want 0x%02x", i, b, expected)
		}
	}
}

func TestQuestFindNamePayloadPreservesFixtureSize(t *testing.T) {
	for _, name := range []string{"0001.dat", "0002.dat"} {
		t.Run(name, func(t *testing.T) {
			raw := readQuestFixture(t, name)
			qf, err := questfile.Read(bytes.NewReader(raw))
			if err != nil {
				t.Fatalf("questfile.Read() error = %v", err)
			}

			apiData := questFileToAPIData(qf)
			if *apiData.Objectives[0].Type != questfile.TypeFIND {
				t.Fatalf("objective 1 type = %d, want FIND", *apiData.Objectives[0].Type)
			}
			if len(apiData.Objectives[0].Name) != 18 {
				t.Fatalf("objective 1 name length = %d, want 18", len(apiData.Objectives[0].Name))
			}

			applyQuestFileAPIData(&qf, apiData)
			var buf bytes.Buffer
			if err := questfile.Write(&buf, qf); err != nil {
				t.Fatalf("questfile.Write() error = %v", err)
			}
			if buf.Len() != 798 {
				t.Fatalf("written file size = %d, want 798", buf.Len())
			}
		})
	}
}

func TestQuestApplyPreservesHeaderPadding(t *testing.T) {
	raw := readQuestFixture(t, "0001.dat")
	qf, err := questfile.Read(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("questfile.Read() error = %v", err)
	}

	originalTargetPadding := append([]byte(nil), qf.Header.TargetNPCBlock[2:]...)
	originalHeaderTail := qf.Header.HeaderTail
	apiData := questFileToAPIData(qf)
	nextExpReward := *apiData.ExpReward + 1
	apiData.ExpReward = &nextExpReward
	applyQuestFileAPIData(&qf, apiData)

	if !bytes.Equal(originalTargetPadding, qf.Header.TargetNPCBlock[2:]) {
		t.Fatal("target NPC padding changed")
	}
	if originalHeaderTail != qf.Header.HeaderTail {
		t.Fatal("header tail padding changed")
	}
}

func TestQuestSampleObjectiveTypes(t *testing.T) {
	raw := readQuestFixture(t, "0412.dat")
	qf, err := questfile.Read(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("questfile.Read() error = %v", err)
	}

	apiData := questFileToAPIData(qf)
	expectedTypes := []uint8{questfile.TypeKILL, questfile.TypeQUESTITEM, questfile.TypeBRINGNPC, questfile.TypeUnused}
	for i, expectedType := range expectedTypes {
		if *apiData.Objectives[i].Type != expectedType {
			t.Fatalf("objective %d type = %d, want %d", i+1, *apiData.Objectives[i].Type, expectedType)
		}
	}

	if binary.LittleEndian.Uint16(qf.Objectives[0].Block[28:30]) != 101 {
		t.Fatal("KILL objective drop item sample did not parse as expected")
	}
}

func TestDropAPIDataRoundTripPreservesBytes(t *testing.T) {
	original := dropfile.DropFile{
		{ItemID: 100, DropRate: 75, DropGroup: 1},
		{ItemID: 0x4001, DropRate: 200, DropGroup: 2},
		{ItemID: dropfile.EmptyItemID},
	}

	var raw bytes.Buffer
	if err := dropfile.Write(&raw, original); err != nil {
		t.Fatalf("dropfile.Write() error = %v", err)
	}

	apiData := dropFileToAPIData(original)
	next := dropFileFromAPIData(apiData)

	var buf bytes.Buffer
	if err := dropfile.Write(&buf, next); err != nil {
		t.Fatalf("dropfile.Write() error = %v", err)
	}

	if !bytes.Equal(raw.Bytes(), buf.Bytes()) {
		t.Fatalf("round-trip changed drop file bytes: before=%d after=%d", raw.Len(), buf.Len())
	}
}

func TestDropFileToAPIDataPreservesFlaggedItemCode(t *testing.T) {
	apiData := dropFileToAPIData(dropfile.DropFile{{ItemID: 0x4001, DropRate: 100, DropGroup: 2}})

	if len(apiData.Drops) != 1 {
		t.Fatalf("drop count = %d, want 1", len(apiData.Drops))
	}
	if *apiData.Drops[0].ItemCode != 0x4001 {
		t.Fatalf("item code = %d, want %d", *apiData.Drops[0].ItemCode, 0x4001)
	}
}

func TestItemCombinationAPIDataRoundTripPreservesBytes(t *testing.T) {
	original := itemcombinationdata.ItemCombinationData{
		{
			Item1:       1,
			Item2:       2,
			Item10:      10,
			SuccessRate: 120,
			Outcome:     500,
			Unknown1:    0x1111,
			Unknown2:    0x2222,
			Unknown3:    0x3333,
			Unknown4:    0x4444,
		},
		{
			Item1:       100,
			SuccessRate: 1,
			Outcome:     600,
			Unknown1:    0xAAAA,
			Unknown2:    0xBBBB,
			Unknown3:    0xCCCC,
			Unknown4:    0xDDDD,
		},
	}

	var raw bytes.Buffer
	if err := itemcombinationdata.Write(&raw, original); err != nil {
		t.Fatalf("itemcombinationdata.Write() error = %v", err)
	}

	apiData := itemCombinationDataToAPIData(original)
	next := itemCombinationDataFromAPIData(apiData, original)

	var buf bytes.Buffer
	if err := itemcombinationdata.Write(&buf, next); err != nil {
		t.Fatalf("itemcombinationdata.Write() error = %v", err)
	}

	if !bytes.Equal(raw.Bytes(), buf.Bytes()) {
		t.Fatalf("round-trip changed item combination bytes: before=%d after=%d", raw.Len(), buf.Len())
	}
}

func TestItemCombinationAPIDataPreservesUnknownsForUpdatedRows(t *testing.T) {
	existing := itemcombinationdata.ItemCombinationData{
		{
			Item1:       1,
			SuccessRate: 50,
			Outcome:     500,
			Unknown1:    0x1111,
			Unknown2:    0x2222,
			Unknown3:    0x3333,
			Unknown4:    0x4444,
		},
	}

	apiData := itemCombinationDataToAPIData(existing)
	*apiData.Formulas[0].Ingredients[0] = 99
	*apiData.Formulas[0].SuccessRate = 75
	*apiData.Formulas[0].Outcome = 600

	next := itemCombinationDataFromAPIData(apiData, existing)
	if next[0].Item1 != 99 || next[0].SuccessRate != 75 || next[0].Outcome != 600 {
		t.Fatal("visible fields were not updated")
	}

	if next[0].Unknown1 != existing[0].Unknown1 ||
		next[0].Unknown2 != existing[0].Unknown2 ||
		next[0].Unknown3 != existing[0].Unknown3 ||
		next[0].Unknown4 != existing[0].Unknown4 {
		t.Fatal("unknown fields changed for existing row")
	}
}

func TestItemCombinationAPIDataDefaultsNewRowUnknownsToZero(t *testing.T) {
	existing := itemcombinationdata.ItemCombinationData{
		{
			Item1:       1,
			SuccessRate: 50,
			Outcome:     500,
			Unknown1:    0x1111,
			Unknown2:    0x2222,
			Unknown3:    0x3333,
			Unknown4:    0x4444,
		},
	}

	apiData := itemCombinationDataToAPIData(existing)
	newSuccessRate := uint16(1)
	newOutcome := uint16(700)
	apiData.Formulas = append(apiData.Formulas, ItemCombinationFormulaAPIData{
		Ingredients: []*uint16{
			uint16Ptr(0),
			uint16Ptr(2),
			uint16Ptr(0),
			uint16Ptr(0),
			uint16Ptr(0),
			uint16Ptr(0),
			uint16Ptr(0),
			uint16Ptr(0),
			uint16Ptr(0),
			uint16Ptr(0),
		},
		SuccessRate: &newSuccessRate,
		Outcome:     &newOutcome,
	})

	next := itemCombinationDataFromAPIData(apiData, existing)
	if len(next) != 2 {
		t.Fatalf("formula count = %d, want 2", len(next))
	}

	if next[1].Unknown1 != 0 || next[1].Unknown2 != 0 || next[1].Unknown3 != 0 || next[1].Unknown4 != 0 {
		t.Fatal("new row unknown fields should default to zero")
	}
}

func TestItemFileAPIDataRoundTripPreservesBytes(t *testing.T) {
	t.Run("it0", func(t *testing.T) {
		original := itemfile.IT0File{{ItemCodeBase: 2, Row: 0, Slot: 3, Type: 1, NPCPrice: 100}}
		copy(original[0].Name[:], "Sword")
		original[0].Name[10] = 0x55
		original[0].Unknown2[0] = 0x1111
		original[0].Levels[0] = itemfile.IT0RawLevelProperties{
			AdditionalAttribute: 1,
			Strength:            2,
			Dexterity:           3,
			Intelligence:        4,
			Attribute:           5,
			Range:               6,
			BlueOption:          7,
			RedOption:           8,
			GreyOption:          9,
		}

		raw, err := serializeIT0ItemFile(original)
		if err != nil {
			t.Fatalf("serializeIT0ItemFile() error = %v", err)
		}

		apiData := it0ItemFileToAPIData(original)
		next, errs := it0ItemFileFromAPIData(apiData, original)
		if len(errs) > 0 {
			t.Fatalf("it0ItemFileFromAPIData() errors = %v", errs)
		}

		nextRaw, err := serializeIT0ItemFile(next)
		if err != nil {
			t.Fatalf("serializeIT0ItemFile(next) error = %v", err)
		}

		if !bytes.Equal(raw, nextRaw) {
			t.Fatal("IT0 round-trip changed bytes")
		}
	})

	t.Run("it1", func(t *testing.T) {
		original := itemfile.IT1File{{Type: 4, Row: 0, NPCPrice: 200, Unknown1: 0x1111, RequiredLevel: 10, Attribute: 20, BlueOption: 30, RedOption: 40, GreyOption: 50}}
		copy(original[0].Name[:], "Potion")
		original[0].Name[10] = 0x55

		raw, err := serializeIT1ItemFile(original)
		if err != nil {
			t.Fatalf("serializeIT1ItemFile() error = %v", err)
		}

		apiData := it1ItemFileToAPIData(original)
		next, errs := it1ItemFileFromAPIData(apiData, original)
		if len(errs) > 0 {
			t.Fatalf("it1ItemFileFromAPIData() errors = %v", errs)
		}

		nextRaw, err := serializeIT1ItemFile(next)
		if err != nil {
			t.Fatalf("serializeIT1ItemFile(next) error = %v", err)
		}

		if !bytes.Equal(raw, nextRaw) {
			t.Fatal("IT1 round-trip changed bytes")
		}
	})

	t.Run("it2", func(t *testing.T) {
		original := itemfile.IT2File{{Type: 5, Row: 0, NPCPrice: 300, Class: 1, RequiredLevel: 20, Unknown1: 0x1111, SkillLevel: 5}}
		copy(original[0].Name[:], "Skill")
		original[0].Name[10] = 0x55

		raw, err := serializeIT2ItemFile(original)
		if err != nil {
			t.Fatalf("serializeIT2ItemFile() error = %v", err)
		}

		apiData := it2ItemFileToAPIData(original)
		next, errs := it2ItemFileFromAPIData(apiData, original)
		if len(errs) > 0 {
			t.Fatalf("it2ItemFileFromAPIData() errors = %v", errs)
		}

		nextRaw, err := serializeIT2ItemFile(next)
		if err != nil {
			t.Fatalf("serializeIT2ItemFile(next) error = %v", err)
		}

		if !bytes.Equal(raw, nextRaw) {
			t.Fatal("IT2 round-trip changed bytes")
		}
	})

	t.Run("it3", func(t *testing.T) {
		original := itemfile.IT3File{{Type: 6, Row: 0, NPCPrice: 400, Unknown1: 0x1111, Unknown2: 0x2222, Unknown3: 0x3333, Unknown4: 0x4444}}
		copy(original[0].Name[:], "Quest Item")
		original[0].Name[15] = 0x55

		raw, err := serializeIT3ItemFile(original)
		if err != nil {
			t.Fatalf("serializeIT3ItemFile() error = %v", err)
		}

		apiData := it3ItemFileToAPIData(original)
		next, errs := it3ItemFileFromAPIData(apiData, original)
		if len(errs) > 0 {
			t.Fatalf("it3ItemFileFromAPIData() errors = %v", errs)
		}

		nextRaw, err := serializeIT3ItemFile(next)
		if err != nil {
			t.Fatalf("serializeIT3ItemFile(next) error = %v", err)
		}

		if !bytes.Equal(raw, nextRaw) {
			t.Fatal("IT3 round-trip changed bytes")
		}
	})
}

func TestIT0ItemFileAPIDataUpdatesVisibleFieldsOnly(t *testing.T) {
	original := itemfile.IT0File{{ItemCodeBase: 2, Row: 0, Slot: 3, Type: 1, NPCPrice: 100}}
	copy(original[0].Name[:], "Sword")
	original[0].Unknown2[0] = 0x1111

	apiData := it0ItemFileToAPIData(original)
	apiData.Items[0].Name = "Blade"
	*apiData.Items[0].NPCPrice = 150
	*apiData.Items[0].Slot = 4
	*apiData.Items[0].Levels[0].Strength = 99

	next, errs := it0ItemFileFromAPIData(apiData, original)
	if len(errs) > 0 {
		t.Fatalf("it0ItemFileFromAPIData() errors = %v", errs)
	}

	if next[0].GetName() != "Blade" || next[0].NPCPrice != 150 || next[0].Slot != 4 || next[0].Levels[0].Strength != 99 {
		t.Fatal("visible IT0 fields were not updated")
	}

	if next[0].ItemCodeBase != original[0].ItemCodeBase || next[0].Row != original[0].Row || next[0].Type != original[0].Type || next[0].Unknown2 != original[0].Unknown2 {
		t.Fatal("read-only or unknown IT0 fields changed")
	}
}

func TestIT0ExItemFileAPIDataValidatesRows(t *testing.T) {
	base := itemfile.IT0File{
		{ItemCodeBase: 2, Row: 0, Slot: 3, Type: 1},
		{ItemCodeBase: 2, Row: 1, Slot: 4, Type: 1},
	}
	copy(base[0].Name[:], "Sword")
	copy(base[1].Name[:], "Axe")

	apiData, err := it0ExItemFileToAPIData(itemfile.IT0ExFile{{Row: 0}}, base)
	if err != nil {
		t.Fatalf("it0ExItemFileToAPIData() error = %v", err)
	}

	if len(apiData.BaseItems) != 2 {
		t.Fatalf("base item count = %d, want 2", len(apiData.BaseItems))
	}
	if *apiData.Items[0].Levels[0].Level != 11 {
		t.Fatalf("first extension level = %d, want 11", *apiData.Items[0].Levels[0].Level)
	}

	row := uint16(1)
	apiData.Items = []ItemFileItemAPIData{{
		Row:    &row,
		Levels: itemFileLevelsToAPIData(make([]itemfile.IT0RawLevelProperties, 5), 11),
	}}
	next, errs := it0ExItemFileFromAPIData(apiData, base)
	if len(errs) > 0 {
		t.Fatalf("it0ExItemFileFromAPIData() errors = %v", errs)
	}
	if len(next) != 1 || next[0].Row != 1 {
		t.Fatal("IT0Ex add/delete update did not preserve requested rows")
	}

	apiData.Items = append(apiData.Items, apiData.Items[0])
	_, errs = it0ExItemFileFromAPIData(apiData, base)
	if len(errs) == 0 {
		t.Fatal("expected duplicate IT0Ex row validation error")
	}

	missingRow := uint16(2)
	apiData.Items = []ItemFileItemAPIData{{
		Row:    &missingRow,
		Levels: itemFileLevelsToAPIData(make([]itemfile.IT0RawLevelProperties, 5), 11),
	}}
	_, errs = it0ExItemFileFromAPIData(apiData, base)
	if len(errs) == 0 {
		t.Fatal("expected missing base row validation error")
	}
}

func TestFixedItemFileAPIDataRejectsCountChanges(t *testing.T) {
	original := itemfile.IT1File{{Type: 4, Row: 0, NPCPrice: 200}}
	apiData := it1ItemFileToAPIData(original)
	apiData.Items = append(apiData.Items, apiData.Items[0])

	_, errs := it1ItemFileFromAPIData(apiData, original)
	if len(errs) == 0 {
		t.Fatal("expected fixed item file count validation error")
	}
}

func TestValidateItemCombinationDataRequest(t *testing.T) {
	successRate := uint16(0)
	outcome := uint16(500)
	req := ItemCombinationDataFileAPIData{
		Formulas: []ItemCombinationFormulaAPIData{
			{
				Ingredients: []*uint16{uint16Ptr(1)},
				SuccessRate: &successRate,
				Outcome:     &outcome,
			},
		},
	}

	errs := validateItemCombinationDataRequest(req)
	if len(errs) != 2 {
		t.Fatalf("validation error count = %d, want 2: %v", len(errs), errs)
	}
}

func questFixtureNames() []string {
	return []string{"0001.dat", "0002.dat", "0412.dat", "1432.dat"}
}

func readQuestFixture(t *testing.T, name string) []byte {
	t.Helper()

	raw, ok := questFixtures[name]
	if !ok {
		t.Fatalf("quest fixture %q not found", name)
	}

	return append([]byte(nil), raw...)
}

func mustDecodeQuestFixture(value string) []byte {
	raw, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		panic(err)
	}

	return raw
}
