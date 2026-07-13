package services

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/project-agonyl/agonyl-utils-go/lotteryfile"
	"github.com/project-agonyl/agonyl-utils-go/npcskillfile"
	"github.com/project-agonyl/agonyl-utils-go/questexfile"
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
	resolvedRoot, err := filepath.EvalSymlinks(root)
	require.NoError(t, err)
	format, ok = service.DetectResolved(resolvedRoot, inside)
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

func TestIsZoneDataCandidatePath(t *testing.T) {
	tests := map[string]bool{
		filepath.Join("ZoneData", "map", "1.map"):               true,
		filepath.Join("ZoneData", "npc", "NPCSkill"):            true,
		filepath.Join("ZoneData", "pc", "0"):                    true,
		filepath.Join("ZoneData", "item", "PresentItemSet.dat"): true,
		filepath.Join("ZoneData", "readme.txt"):                 false,
		filepath.Join("logs", "zone.log"):                       false,
	}

	for path, expected := range tests {
		require.Equal(t, expected, IsZoneDataCandidatePath(path), path)
	}
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

func TestZoneDataDetectsV07BinaryFormats(t *testing.T) {
	root := t.TempDir()
	tests := []struct {
		format ZoneDataFormat
		parts  []string
	}{
		{ZoneDataFormatMessage, []string{"A3Msg_Zone_Tw.dat"}},
		{ZoneDataFormatQuestEx, []string{"ZoneData", "quest", "0042.dat"}},
		{ZoneDataFormatSQuestQuiz, []string{"ZoneData", "SQuest", "QuizTable.dat"}},
		{ZoneDataFormatTowerTreasure, []string{"Tower", "6.itm"}},
		{ZoneDataFormatOXQuiz, []string{"OXQuiz", "OXQuizTable.dat"}},
		{ZoneDataFormatTyrBase, []string{"ZoneData", "tyr", "BaseInfo.tyr"}},
		{ZoneDataFormatTyrPortal, []string{"ZoneData", "tyr", "WarpPortal.tyr"}},
		{ZoneDataFormatTyrUpgrade, []string{"ZoneData", "tyr", "Upgrade.tyr"}},
		{ZoneDataFormatTyrStartPoint, []string{"ZoneData", "tyr", "StartPoint.tyr"}},
		{ZoneDataFormatTyrGift, []string{"ZoneData", "tyr", "TyrGift.dat"}},
		{ZoneDataFormatTyrNPCRegen, []string{"ZoneData", "tyr", "NPCRegen.tyr"}},
		{ZoneDataFormatTyrSkillLayer, []string{"ZoneData", "tyr", "SkillLayer.tyr"}},
	}
	service := &zoneDataService{}
	for _, test := range tests {
		path := writeZoneDataTestFile(t, root, test.parts...)
		format, ok := service.Detect(root, path)
		require.True(t, ok, path)
		require.Equal(t, test.format, format, path)
	}
}

func TestZoneDataApplyPreservesQuestExOpaqueBytes(t *testing.T) {
	var data questexfile.Data
	require.NoError(t, data.SetInt32(0, 42))
	require.NoError(t, data.SetInt32(4, 1000))
	var original bytes.Buffer
	require.NoError(t, questexfile.Write(&original, data))
	operation := ZoneDataOperation{Scope: "row", Row: 0, Field: "start_npc", Value: json.RawMessage("2000")}

	updated, err := (&zoneDataService{}).Apply(original.Bytes(), ZoneDataFormatQuestEx, []ZoneDataOperation{operation})
	require.NoError(t, err)
	require.Equal(t, original.Bytes()[:4], updated[:4])
	require.Equal(t, original.Bytes()[8:], updated[8:])
}

func writeZoneDataTestFile(t *testing.T, root string, parts ...string) string {
	t.Helper()

	path := filepath.Join(append([]string{root}, parts...)...)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, nil, 0600))
	return path
}
