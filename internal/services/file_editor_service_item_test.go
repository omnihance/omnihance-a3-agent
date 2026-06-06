package services

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/omnihance/omnihance-a3-agent/internal/logger"
	"github.com/project-agonyl/agonyl-utils-go/dropfile"
	"github.com/project-agonyl/agonyl-utils-go/itemcombinationdata"
	"github.com/project-agonyl/agonyl-utils-go/itemfile"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestReadClientItemFileBytes(t *testing.T) {
	log := logger.NewZerologLogger(zerolog.Nop(), "test", zerolog.Disabled)
	service := NewFileEditorService(log)

	tests := []struct {
		name     string
		write    func(*bytes.Buffer) error
		read     func([]byte) ([]itemfile.Item, error)
		expected itemfile.Item
	}{
		{
			name: "it0",
			write: func(buf *bytes.Buffer) error {
				data := itemfile.IT0File{{ItemCodeBase: 2, Row: 0, Slot: 3, Type: 1, NPCPrice: 100}}
				copy(data[0].Name[:], "Sword")
				return itemfile.WriteIT0(buf, data)
			},
			read: service.ReadClientIT0FileBytes,
			expected: itemfile.Item{
				ItemCode:  2048,
				SlotIndex: 3,
				ItemName:  "Sword",
				Itemtype:  1,
				NPCPrice:  100,
			},
		},
		{
			name: "it1",
			write: func(buf *bytes.Buffer) error {
				data := itemfile.IT1File{{Type: 4, Row: 0, NPCPrice: 200}}
				copy(data[0].Name[:], "Potion")
				return itemfile.WriteIT1(buf, data)
			},
			read: service.ReadClientIT1FileBytes,
			expected: itemfile.Item{
				ItemCode:  4096,
				SlotIndex: 8,
				ItemName:  "Potion",
				Itemtype:  4,
				NPCPrice:  200,
			},
		},
		{
			name: "it2",
			write: func(buf *bytes.Buffer) error {
				data := itemfile.IT2File{{Type: 5, Row: 0, NPCPrice: 300, Class: 1}}
				copy(data[0].Name[:], "Skill")
				return itemfile.WriteIT2(buf, data)
			},
			read: service.ReadClientIT2FileBytes,
			expected: itemfile.Item{
				ItemCode: 5120,
				ItemName: "Skill",
				Itemtype: 5,
				NPCPrice: 300,
			},
		},
		{
			name: "it3",
			write: func(buf *bytes.Buffer) error {
				data := itemfile.IT3File{{Type: 6, Row: 0, NPCPrice: 400}}
				copy(data[0].Name[:], "Quest Item")
				return itemfile.WriteIT3(buf, data)
			},
			read: service.ReadClientIT3FileBytes,
			expected: itemfile.Item{
				ItemCode: 6144,
				ItemName: "Quest Item",
				Itemtype: 6,
				NPCPrice: 400,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buf bytes.Buffer
			require.NoError(t, test.write(&buf))

			items, err := test.read(buf.Bytes())
			require.NoError(t, err)
			require.Len(t, items, 1)
			require.Equal(t, test.expected.ItemCode, items[0].ItemCode)
			require.Equal(t, test.expected.SlotIndex, items[0].SlotIndex)
			require.Equal(t, test.expected.ItemName, items[0].ItemName)
			require.Equal(t, test.expected.Itemtype, items[0].Itemtype)
			require.Equal(t, test.expected.NPCPrice, items[0].NPCPrice)
		})
	}
}

func TestReadClientItemFileBytesTruncated(t *testing.T) {
	log := logger.NewZerologLogger(zerolog.Nop(), "test", zerolog.Disabled)
	service := NewFileEditorService(log)

	_, err := service.ReadClientIT1FileBytes([]byte{1, 2, 3})
	require.Error(t, err)
}

func TestGetFileTypeItemFiles(t *testing.T) {
	log := logger.NewZerologLogger(zerolog.Nop(), "test", zerolog.Disabled)
	service := NewFileEditorService(log)
	dir := t.TempDir()

	tests := []struct {
		name     string
		fileType FileType
	}{
		{name: "0", fileType: FileTypeIT0Item},
		{name: "1", fileType: FileTypeIT1Item},
		{name: "2", fileType: FileTypeIT2Item},
		{name: "3", fileType: FileTypeIT3Item},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(dir, test.name)
			require.NoError(t, os.WriteFile(path, nil, 0644))
			info, err := os.Stat(path)
			require.NoError(t, err)

			require.Equal(t, test.fileType, service.GetFileType(path, info))
			require.True(t, service.IsFileViewable(path, info))
			require.True(t, service.IsFileEditable(path, info))
			require.Equal(t, "/file-tree/item-file", service.GetFileAPIEndpoint(path, info))
		})
	}

	t.Run("0ex requires sibling 0", func(t *testing.T) {
		exPath := filepath.Join(t.TempDir(), "0ex")
		require.NoError(t, os.WriteFile(exPath, nil, 0644))
		info, err := os.Stat(exPath)
		require.NoError(t, err)

		require.Equal(t, FileTypeIT0ExItem, service.GetFileType(exPath, info))
		require.False(t, service.IsFileViewable(exPath, info))
		require.False(t, service.IsFileEditable(exPath, info))

		basePath := filepath.Join(filepath.Dir(exPath), "0")
		require.NoError(t, os.Mkdir(basePath, 0755))
		require.False(t, service.IsFileViewable(exPath, info))
		require.False(t, service.IsFileEditable(exPath, info))

		require.NoError(t, os.Remove(basePath))
		require.NoError(t, os.WriteFile(basePath, nil, 0644))
		require.True(t, service.IsFileViewable(exPath, info))
		require.True(t, service.IsFileEditable(exPath, info))
		require.Equal(t, "/file-tree/item-file", service.GetFileAPIEndpoint(exPath, info))
	})
}

func TestRawItemFileDataRoundTrip(t *testing.T) {
	log := logger.NewZerologLogger(zerolog.Nop(), "test", zerolog.Disabled)
	service := NewFileEditorService(log)
	dir := t.TempDir()

	t.Run("it0", func(t *testing.T) {
		path := filepath.Join(dir, "0")
		expected := itemfile.IT0File{{ItemCodeBase: 2, Row: 0, Slot: 3, Type: 1, NPCPrice: 100}}
		copy(expected[0].Name[:], "Sword")
		expected[0].Unknown2[0] = 0x1111
		expected[0].Levels[0].Strength = 10

		require.NoError(t, service.WriteIT0ItemFileData(path, expected))
		actual, err := service.ReadIT0ItemFileData(path)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("it0ex", func(t *testing.T) {
		path := filepath.Join(dir, "0ex")
		expected := itemfile.IT0ExFile{{Row: 0}}
		expected[0].Levels[0].Strength = 10

		require.NoError(t, service.WriteIT0ExItemFileData(path, expected))
		actual, err := service.ReadIT0ExItemFileData(path)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("it1", func(t *testing.T) {
		path := filepath.Join(dir, "1")
		expected := itemfile.IT1File{{Type: 4, Row: 0, NPCPrice: 200, Unknown1: 0x1111, RequiredLevel: 10}}
		copy(expected[0].Name[:], "Potion")

		require.NoError(t, service.WriteIT1ItemFileData(path, expected))
		actual, err := service.ReadIT1ItemFileData(path)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("it2", func(t *testing.T) {
		path := filepath.Join(dir, "2")
		expected := itemfile.IT2File{{Type: 5, Row: 0, NPCPrice: 300, Class: 1, Unknown1: 0x1111, SkillLevel: 5}}
		copy(expected[0].Name[:], "Skill")

		require.NoError(t, service.WriteIT2ItemFileData(path, expected))
		actual, err := service.ReadIT2ItemFileData(path)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})

	t.Run("it3", func(t *testing.T) {
		path := filepath.Join(dir, "3")
		expected := itemfile.IT3File{{Type: 6, Row: 0, NPCPrice: 400, Unknown1: 0x1111, Unknown2: 0x2222}}
		copy(expected[0].Name[:], "Quest Item")

		require.NoError(t, service.WriteIT3ItemFileData(path, expected))
		actual, err := service.ReadIT3ItemFileData(path)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
	})
}

func TestReadClientMonsterAndMapFileBytesRejectInvalidInput(t *testing.T) {
	log := logger.NewZerologLogger(zerolog.Nop(), "test", zerolog.Disabled)
	service := NewFileEditorService(log)

	tests := []struct {
		name string
		read func([]byte) error
	}{
		{
			name: "monster",
			read: func(data []byte) error {
				_, err := service.ReadClientMonsterFileBytes(data)
				return err
			},
		},
		{
			name: "map",
			read: func(data []byte) error {
				_, err := service.ReadClientMapFileBytes(data)
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name+" empty", func(t *testing.T) {
			require.ErrorIs(t, test.read(nil), io.ErrUnexpectedEOF)
		})

		t.Run(test.name+" short header", func(t *testing.T) {
			require.ErrorIs(t, test.read([]byte{1, 2, 3}), io.ErrUnexpectedEOF)
		})

		t.Run(test.name+" truncated body", func(t *testing.T) {
			data := make([]byte, clientDataHeaderSize+1)
			binary.LittleEndian.PutUint32(data[:clientDataHeaderSize], 1)
			require.ErrorIs(t, test.read(data), io.ErrUnexpectedEOF)
		})

		t.Run(test.name+" oversized count", func(t *testing.T) {
			data := make([]byte, clientDataHeaderSize)
			binary.LittleEndian.PutUint32(data, ^uint32(0))
			require.ErrorIs(t, test.read(data), io.ErrUnexpectedEOF)
		})
	}

	t.Run("monster valid", func(t *testing.T) {
		data := make([]byte, clientDataHeaderSize+96)
		binary.LittleEndian.PutUint32(data[:clientDataHeaderSize], 1)

		monsters, err := service.ReadClientMonsterFileBytes(data)
		require.NoError(t, err)
		require.Len(t, monsters, 1)
	})

	t.Run("map valid", func(t *testing.T) {
		data := make([]byte, clientDataHeaderSize+56)
		binary.LittleEndian.PutUint32(data[:clientDataHeaderSize], 1)

		maps, err := service.ReadClientMapFileBytes(data)
		require.NoError(t, err)
		require.Len(t, maps, 1)
	})
}

func TestDropFileDataRoundTrip(t *testing.T) {
	log := logger.NewZerologLogger(zerolog.Nop(), "test", zerolog.Disabled)
	service := NewFileEditorService(log)
	path := filepath.Join(t.TempDir(), "100.itm")

	expected := dropfile.DropFile{
		{ItemID: 100, DropRate: 75, DropGroup: 1},
		{ItemID: 0x4001, DropRate: 200, DropGroup: 2},
		{ItemID: dropfile.EmptyItemID},
	}

	require.NoError(t, service.WriteDropFileData(path, expected))
	actual, err := service.ReadDropFileData(path)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestReadDropFileDataEmpty(t *testing.T) {
	log := logger.NewZerologLogger(zerolog.Nop(), "test", zerolog.Disabled)
	service := NewFileEditorService(log)
	path := filepath.Join(t.TempDir(), "empty.itm")
	require.NoError(t, os.WriteFile(path, nil, 0644))

	actual, err := service.ReadDropFileData(path)
	require.NoError(t, err)
	require.NotNil(t, actual)
	require.Empty(t, actual)
}

func TestReadDropFileDataTruncated(t *testing.T) {
	log := logger.NewZerologLogger(zerolog.Nop(), "test", zerolog.Disabled)
	service := NewFileEditorService(log)
	path := filepath.Join(t.TempDir(), "bad.itm")
	require.NoError(t, os.WriteFile(path, []byte{1, 2, 3}, 0644))

	_, err := service.ReadDropFileData(path)
	require.Error(t, err)
}

func TestGetFileTypeItemCombinationData(t *testing.T) {
	log := logger.NewZerologLogger(zerolog.Nop(), "test", zerolog.Disabled)
	service := NewFileEditorService(log)
	path := filepath.Join(t.TempDir(), "itemcombinationdata")
	require.NoError(t, os.WriteFile(path, nil, 0644))

	info, err := os.Stat(path)
	require.NoError(t, err)
	require.Equal(t, FileTypeItemCombinationData, service.GetFileType(path, info))
	require.True(t, service.IsFileViewable(path, info))
	require.True(t, service.IsFileEditable(path, info))
	require.Equal(t, "/file-tree/item-combination-data", service.GetFileAPIEndpoint(path, info))
}

func TestItemCombinationDataRoundTrip(t *testing.T) {
	log := logger.NewZerologLogger(zerolog.Nop(), "test", zerolog.Disabled)
	service := NewFileEditorService(log)
	path := filepath.Join(t.TempDir(), "ItemCombinationData")

	expected := itemcombinationdata.ItemCombinationData{
		{
			Item1:       1,
			Item2:       2,
			Item3:       0,
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
			Item2:       101,
			SuccessRate: 1,
			Outcome:     600,
		},
	}

	require.NoError(t, service.WriteItemCombinationData(path, expected))
	actual, err := service.ReadItemCombinationData(path)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestReadItemCombinationDataEmpty(t *testing.T) {
	log := logger.NewZerologLogger(zerolog.Nop(), "test", zerolog.Disabled)
	service := NewFileEditorService(log)
	path := filepath.Join(t.TempDir(), "ItemCombinationData")
	require.NoError(t, os.WriteFile(path, nil, 0644))

	actual, err := service.ReadItemCombinationData(path)
	require.NoError(t, err)
	require.NotNil(t, actual)
	require.Empty(t, actual)
}

func TestReadItemCombinationDataTruncated(t *testing.T) {
	log := logger.NewZerologLogger(zerolog.Nop(), "test", zerolog.Disabled)
	service := NewFileEditorService(log)
	path := filepath.Join(t.TempDir(), "ItemCombinationData")
	require.NoError(t, os.WriteFile(path, []byte{1, 2, 3}, 0644))

	_, err := service.ReadItemCombinationData(path)
	require.Error(t, err)
}
