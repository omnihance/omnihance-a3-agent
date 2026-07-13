package services

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/project-agonyl/agonyl-utils-go/lotteryfile"
	"github.com/project-agonyl/agonyl-utils-go/npcskillfile"
	"github.com/stretchr/testify/require"
)

func TestZoneDataDetectRequiresConfiguredRootContainment(t *testing.T) {
	root := t.TempDir()
	inside := writeZoneDataTestFile(t, root, "ZoneData", "pc", "0")
	outside := writeZoneDataTestFile(t, t.TempDir(), "ZoneData", "pc", "0")
	service := &zoneDataService{}

	format, ok := service.Detect(root, inside)
	require.True(t, ok)
	require.Equal(t, ZoneDataFormatPCData, format)

	_, ok = service.Detect(root, outside)
	require.False(t, ok)
}

func TestZoneDataDetectUsesRelativePathPrecedence(t *testing.T) {
	root := t.TempDir()
	pc := writeZoneDataTestFile(t, root, "ZoneData", "pc", "0")
	skill := writeZoneDataTestFile(t, root, "ZoneData", "skill", "0")
	item := writeZoneDataTestFile(t, root, "ZoneData", "item", "0")
	service := &zoneDataService{}

	format, ok := service.Detect(root, pc)
	require.True(t, ok)
	require.Equal(t, ZoneDataFormatPCData, format)
	format, ok = service.Detect(root, skill)
	require.True(t, ok)
	require.Equal(t, ZoneDataFormatSkillData, format)
	_, ok = service.Detect(root, item)
	require.False(t, ok)
}

func TestZoneDataApplyPreservesOpaqueNPCSkillBytes(t *testing.T) {
	original := make([]byte, npcskillfile.RecordSize)
	for index := range original {
		original[index] = byte(index + 1)
	}
	service := &zoneDataService{}
	operation := ZoneDataOperation{Scope: "row", Row: 0, Field: "effect_code", Value: json.RawMessage("48879")}

	updated, err := service.Apply(original, ZoneDataFormatNPCSkill, []ZoneDataOperation{operation})
	require.NoError(t, err)
	require.Equal(t, original[:14], updated[:14])
	require.Equal(t, []byte{0xef, 0xbe}, updated[14:16])
	require.Equal(t, original[16:], updated[16:])

	decoded, err := npcskillfile.Read(bytes.NewReader(updated))
	require.NoError(t, err)
	require.Equal(t, uint16(0xbeef), decoded[0].EffectCode())
}

func TestZoneDataApplyRejectsUnregisteredField(t *testing.T) {
	original := make([]byte, npcskillfile.RecordSize)
	service := &zoneDataService{}
	operation := ZoneDataOperation{Scope: "row", Row: 0, Field: "raw", Value: json.RawMessage("1")}

	_, err := service.Apply(original, ZoneDataFormatNPCSkill, []ZoneDataOperation{operation})
	require.Error(t, err)
}

func TestZoneDataDetectsV06FormatsInsideServerRoot(t *testing.T) {
	root := t.TempDir()
	tests := []struct {
		format ZoneDataFormat
		parts  []string
	}{
		{ZoneDataFormatCashItem, []string{"ZoneData", "shop", "CashItemTbl.dat"}},
		{ZoneDataFormatSetItem, []string{"ZoneData", "item", "SIT2"}},
		{ZoneDataFormatPresentItemSet, []string{"ZoneData", "item", "PresentItemSet.dat"}},
		{ZoneDataFormatPet, []string{"ZoneData", "item", "pet"}},
		{ZoneDataFormatShueCombination, []string{"ZoneData", "item", "ShueCombinationData"}},
		{ZoneDataFormatDerbyGift, []string{"ZoneData", "npc", "DerbyGift.dat"}},
		{ZoneDataFormatLottery, []string{"Event", "LotteryItem.dat"}},
		{ZoneDataFormatEventItemReward, []string{"Event", "EventItem2.dat"}},
		{ZoneDataFormatA3Present, []string{"Present_3.dat"}},
	}
	service := &zoneDataService{}
	for _, test := range tests {
		path := writeZoneDataTestFile(t, root, test.parts...)
		format, ok := service.Detect(root, path)
		require.True(t, ok, path)
		require.Equal(t, test.format, format, path)
	}
}

func TestZoneDataApplyPreservesLotterySiblingMessages(t *testing.T) {
	original := make([]byte, lotteryfile.RecordSize)
	for index := range original {
		original[index] = byte(index)
	}
	operation := ZoneDataOperation{Scope: "row", Row: 0, Field: "message_2", Value: json.RawMessage(`"winner"`)}

	updated, err := (&zoneDataService{}).Apply(original, ZoneDataFormatLottery, []ZoneDataOperation{operation})
	require.NoError(t, err)
	offset := 4 + 2*lotteryfile.MessageSize
	require.Equal(t, original[:offset], updated[:offset])
	require.Equal(t, original[offset+lotteryfile.MessageSize:], updated[offset+lotteryfile.MessageSize:])
}

func writeZoneDataTestFile(t *testing.T, root string, parts ...string) string {
	t.Helper()

	path := filepath.Join(append([]string{root}, parts...)...)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, nil, 0600))
	return path
}
