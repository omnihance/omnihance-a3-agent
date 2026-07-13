package services

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

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

func writeZoneDataTestFile(t *testing.T, root string, parts ...string) string {
	t.Helper()

	path := filepath.Join(append([]string{root}, parts...)...)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	require.NoError(t, os.WriteFile(path, nil, 0600))
	return path
}
