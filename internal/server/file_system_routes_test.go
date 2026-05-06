package server

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/project-agonyl/agonyl-utils-go/questfile"
)

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

func questFixtureNames() []string {
	return []string{"0001.dat", "0002.dat", "0412.dat", "1432.dat"}
}

func readQuestFixture(t *testing.T, name string) []byte {
	t.Helper()

	path := filepath.Join("..", "..", "quests", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%s) error = %v", path, err)
	}

	return raw
}
