package server

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"testing"

	"github.com/project-agonyl/agonyl-utils-go/dropfile"
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
